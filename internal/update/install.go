package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// most is the largest download accepted, so a wrong URL cannot fill the
// disk. A sameway binary is a few tens of megabytes.
const most = 300 << 20

// install downloads the release for this machine, checks it against the
// sha256 published with it, and puts it where the running binary is. It
// says which file it replaced.
func (u Updater) install(ctx context.Context, rel *Release) (string, error) {
	if !Known(u.current()) {
		return "", fmt.Errorf("this build says %q, so it will not replace itself with sameway %s; install that release yourself from %s", u.current(), rel.Version, rel.Page)
	}
	if rel.Asset == "" {
		return "", fmt.Errorf("sameway %s publishes nothing for %s/%s; install it yourself from %s", rel.Version, runtime.GOOS, runtime.GOARCH, rel.Page)
	}
	if rel.Sums == "" {
		return "", fmt.Errorf("sameway %s publishes no %s, so the download cannot be checked and will not be installed; install it yourself from %s", rel.Version, sumsFile, rel.Page)
	}
	exe, err := u.binary()
	if err != nil {
		return "", err
	}
	want, err := u.sum(ctx, rel)
	if err != nil {
		return "", err
	}
	data, err := u.fetch(ctx, rel.Asset)
	if err != nil {
		return "", err
	}
	if got := sha256.Sum256(data); got != want {
		return "", fmt.Errorf("%s did not match the sha256 published for it (%s, not %s); nothing was installed", rel.Name, hex.EncodeToString(got[:]), hex.EncodeToString(want[:]))
	}
	prog, err := program(rel.Name, data)
	if err != nil {
		return "", err
	}
	if err := swap(exe, prog); err != nil {
		return "", err
	}
	return exe, nil
}

// binary is the file to replace: the running program, unless a caller
// named another. A program built by `go run` lives in a temporary folder
// and is gone at the next start, so there is nothing there to replace.
func (u Updater) binary() (string, error) {
	if u.Exe != "" {
		return u.Exe, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not find the running program to replace: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if within(exe, os.TempDir()) {
		return "", fmt.Errorf("this sameway is running from %s, so it was built by `go run` and there is nothing lasting to replace; build it with `go build -o bin/sameway ./cmd/sameway` first", filepath.Dir(exe))
	}
	return exe, nil
}

// sum is the sha256 the release publishes for the file being downloaded.
func (u Updater) sum(ctx context.Context, rel *Release) ([32]byte, error) {
	var want [32]byte
	list, err := u.fetch(ctx, rel.Sums)
	if err != nil {
		return want, err
	}
	for _, line := range strings.Split(string(list), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 {
			continue
		}
		// `shasum -a 256` writes "*name" for a binary file.
		if !strings.EqualFold(strings.TrimPrefix(fields[1], "*"), rel.Name) {
			continue
		}
		raw, err := hex.DecodeString(fields[0])
		if err != nil || len(raw) != len(want) {
			return want, fmt.Errorf("%s gives %q as the sha256 of %s, which is not one", sumsFile, fields[0], rel.Name)
		}
		copy(want[:], raw)
		return want, nil
	}
	return want, fmt.Errorf("%s of sameway %s does not list %s, so the download cannot be checked", sumsFile, rel.Version, rel.Name)
}

// fetch reads one release file whole, up to the size accepted.
func (u Updater) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "sameway/"+u.current())
	resp, err := u.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s", url, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, most+1))
	if err != nil {
		return nil, fmt.Errorf("could not download %s: %w", url, err)
	}
	if len(data) > most {
		return nil, fmt.Errorf("%s is larger than %d bytes, which is more than a sameway binary; nothing was installed", url, most)
	}
	return data, nil
}

// program is the executable inside a download: the download itself when it
// is the program, or the one program inside a zip or a tar.gz.
func program(name string, data []byte) ([]byte, error) {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return fromZip(name, data)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return fromTar(name, data)
	}
	return data, nil
}

func fromZip(name string, data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%s is not a zip that can be read: %w", name, err)
	}
	for _, f := range r.File {
		if !isProgram(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%s: could not read %s: %w", name, f.Name, err)
		}
		defer rc.Close()
		return io.ReadAll(io.LimitReader(rc, most))
	}
	return nil, fmt.Errorf("%s holds no sameway program", name)
}

func fromTar(name string, data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%s is not a gzip that can be read: %w", name, err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: could not read the tar: %w", name, err)
		}
		if h.Typeflag == tar.TypeReg && isProgram(h.Name) {
			return io.ReadAll(io.LimitReader(tr, most))
		}
	}
	return nil, fmt.Errorf("%s holds no sameway program", name)
}

// isProgram says whether a file inside an archive is the sameway binary,
// by the only name it is published under.
func isProgram(name string) bool {
	base := strings.ToLower(path.Base(filepath.ToSlash(name)))
	return base == "sameway" || base == "sameway.exe"
}

// swap puts a new program where the running one is. The new file is
// written beside it, so the move is on one volume, and the old one is
// moved aside rather than deleted: Windows will not delete a program that
// is running, but it will rename it. Tidy clears what is left at the next
// start. A move that fails part way puts the old program back.
func swap(exe string, prog []byte) error {
	next, old := exe+".new", exe+".old"
	os.Remove(next)
	if err := os.WriteFile(next, prog, 0o755); err != nil {
		return fmt.Errorf("could not write the new sameway beside %s: %w", exe, err)
	}
	if err := os.Chmod(next, 0o755); err != nil {
		os.Remove(next)
		return fmt.Errorf("could not make the new sameway runnable: %w", err)
	}
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(next)
		return fmt.Errorf("could not move %s aside: %w", exe, err)
	}
	if err := os.Rename(next, exe); err != nil {
		os.Rename(old, exe)
		os.Remove(next)
		return fmt.Errorf("could not put the new sameway at %s: %w", exe, err)
	}
	os.Remove(old)
	return nil
}

// Tidy removes what an install left behind, which on Windows is the old
// program: it could not be deleted while it was running. exe empty means
// the running program. Nothing here is worth an error.
func Tidy(exe string) {
	if exe == "" {
		if found, err := os.Executable(); err == nil {
			exe = found
		} else {
			return
		}
	}
	os.Remove(exe + ".old")
	os.Remove(exe + ".new")
}

// within says whether a path sits inside a folder.
func within(p, dir string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
