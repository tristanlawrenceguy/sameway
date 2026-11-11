package records

import (
	"regexp"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

var pictureLink = regexp.MustCompile(`/t/file/([a-z0-9]{16})`)

// PictureOf is the picture a record is seen by in a gallery, and the
// words for someone who cannot see it: a picture kept is itself, made
// small (media/small.go), a video is itself, which the gallery shows at
// its first frame, and a note a photo was shared into the first picture
// it links to. The words are the file's description, which the assistant
// writes when it reads a picture. None when it has no picture.
func PictureOf(st *store.Store, t *schema.Type, rec *store.Record) (src, alt string, video bool) {
	if t.Name == FileType {
		return fileSeen(rec)
	}
	for _, f := range t.Fields {
		v, _ := rec.Fields[f.Name].(string)
		for _, m := range pictureLink.FindAllStringSubmatch(v, -1) {
			if file, err := st.Get(FileType, m[1]); err == nil {
				if src, alt, video := fileSeen(file); src != "" {
					return src, alt, video
				}
			}
		}
	}
	return "", "", false
}

func fileSeen(f *store.Record) (src, alt string, video bool) {
	alt, _ = f.Fields["description"].(string)
	switch f.Fields["kind"] {
	case "image":
		return "/files/" + f.ID + "/small", alt, false
	case "video":
		return "/files/" + f.ID, alt, true
	}
	return "", "", false
}
