package speech

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Dir is where speech-to-text is kept on this computer: the user's cache,
// so every workspace shares one copy and nothing lands in a workspace.
func Dir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sameway", "speech", EngineVersion), nil
}

// Progress is told, as a download goes, how much of the whole has come.
type Progress func(done, total int64)

// Install fetches the engine and the model into dir, checks each against
// its fingerprint, and unpacks the engine. A file already there with the
// right fingerprint is not fetched again; one that does not match is
// never kept.
func Install(ctx context.Context, client *http.Client, dir string, progress Progress) error {
	eng, ok := engines[system()]
	if !ok {
		return fmt.Errorf("there is no speech-to-text engine for %s", system())
	}
	return install(ctx, client, dir, eng, model, progress)
}

// install is Install with the files to fetch given.
func install(ctx context.Context, client *http.Client, dir string, eng asset, model []asset, progress Progress) error {
	if client == nil {
		client = http.DefaultClient
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	total, done := eng.Size, int64(0)
	for _, a := range model {
		total += a.Size
	}
	_, err := os.Stat(runner(dir))
	unpacked := err == nil
	all := append([]asset{eng}, model...)
	for _, a := range all {
		path := filepath.Join(dir, a.Name)
		if (a == eng && unpacked) || sameFile(path, a.SHA256) {
			done += a.Size
			continue
		}
		base := done
		err := fetch(ctx, client, a, path, func(n int64) {
			if progress != nil {
				progress(base+n, total)
			}
		})
		if err != nil {
			return err
		}
		done += a.Size
	}
	engineDir := filepath.Join(dir, "engine")
	if unpacked {
		return nil
	}
	os.RemoveAll(engineDir)
	if err := unpack(filepath.Join(dir, eng.Name), engineDir); err != nil {
		os.RemoveAll(engineDir)
		return fmt.Errorf("could not unpack the speech engine: %w", err)
	}
	if _, err := os.Stat(runner(dir)); err != nil {
		return errors.New("the speech engine was unpacked but its program is not in it")
	}
	// Unpacked, the archive is not needed again.
	os.Remove(filepath.Join(dir, eng.Name))
	return nil
}

// Ready says whether speech-to-text is here and whole.
func Ready(dir string) bool {
	if _, err := os.Stat(runner(dir)); err != nil {
		return false
	}
	for _, a := range model {
		if st, err := os.Stat(filepath.Join(dir, a.Name)); err != nil || st.Size() != a.Size {
			return false
		}
	}
	return true
}

// sameFile says whether the file at path has this fingerprint.
func sameFile(path, sum string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == sum
}

// fetch downloads one asset beside path, checks it, and only then puts it
// in place.
func fetch(ctx context.Context, client *http.Client, a asset, path string, progress func(int64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach %s: %w", host(a.URL), err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s for %s", host(a.URL), res.Status, filepath.Base(a.URL))
	}
	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), &counting{r: res.Body, tell: progress})
	f.Close()
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("the download from %s stopped: %w", host(a.URL), err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != a.SHA256 {
		os.Remove(tmp)
		return fmt.Errorf("%s from %s is not the file sameway expects (%d bytes, fingerprint %s…), so it was not kept", filepath.Base(a.URL), host(a.URL), n, got[:12])
	}
	return os.Rename(tmp, path)
}

func host(u string) string {
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	h, _, _ := strings.Cut(u, "/")
	return h
}

// counting tells how much has been read so far.
type counting struct {
	r    io.Reader
	n    int64
	tell func(int64)
}

func (c *counting) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.tell != nil {
		c.tell(c.n)
	}
	return n, err
}

// unpack writes a .tar.bz2 or .tar.gz into dir, dropping its top folder,
// and refuses any entry that would land outside dir.
func unpack(archive, dir string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(archive, ".gz") || strings.HasSuffix(archive, ".tgz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		r = gz
	} else {
		r = bzip2.NewReader(f)
	}
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.ToSlash(h.Name)
		if _, rest, ok := strings.Cut(name, "/"); ok {
			name = rest
		}
		if name == "" {
			continue
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		if rel, err := filepath.Rel(dir, target); err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("the archive has an entry outside itself: %s", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if h.FileInfo().Mode()&0o111 != 0 {
				mode = 0o755
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			if err != nil {
				return err
			}
		case tar.TypeSymlink:
			// Libraries link to their versioned names; the link stays inside.
			if strings.Contains(h.Linkname, "..") || filepath.IsAbs(h.Linkname) {
				continue
			}
			os.Remove(target)
			if err := os.Symlink(h.Linkname, target); err != nil {
				return err
			}
		}
	}
}
