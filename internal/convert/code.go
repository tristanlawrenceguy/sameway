package convert

// Plain text of every other kind is read too: a program's source, a
// configuration, a query, as a code block in its language, so search finds
// its words and the assistant reads it. A .env file is not among them: it
// usually holds secrets, and is kept as it is. Subtitles (.srt, .vtt) are
// read as a transcript, a line per cue with its time.

var codeLanguages = map[string]string{
	"py": "python", "js": "javascript", "mjs": "javascript", "cjs": "javascript", "jsx": "jsx",
	"ts": "typescript", "tsx": "tsx", "go": "go", "rs": "rust", "java": "java", "kt": "kotlin",
	"swift": "swift", "c": "c", "h": "c", "cpp": "cpp", "cc": "cpp", "hpp": "cpp", "cs": "csharp",
	"rb": "ruby", "php": "php", "lua": "lua", "r": "r", "pl": "perl", "scala": "scala", "dart": "dart",
	"sh": "bash", "bash": "bash", "zsh": "bash", "fish": "fish", "ps1": "powershell", "bat": "bat", "cmd": "bat",
	"sql": "sql", "graphql": "graphql", "gql": "graphql", "proto": "protobuf",
	"xml": "xml", "toml": "toml", "ini": "ini", "cfg": "ini", "conf": "ini", "properties": "properties",
	"css": "css", "scss": "scss", "less": "less", "vue": "vue", "svelte": "svelte",
	"tex": "latex", "diff": "diff", "patch": "diff", "ipynb": "json",
}

func init() {
	for ext, lang := range codeLanguages {
		if _, taken := kinds[ext]; !taken {
			kinds[ext] = struct {
				name string
				read func([]byte) (string, error)
			}{"code", code(lang)}
		}
	}
	kinds["ics"] = struct {
		name string
		read func([]byte) (string, error)
	}{"calendar", icsMarkdown}
	kinds["srt"] = struct {
		name string
		read func([]byte) (string, error)
	}{"captions", captions}
	kinds["vtt"] = kinds["srt"]
}

// captions reads subtitles as a transcript.
func captions(data []byte) (string, error) {
	cues := ParseVTT(string(data))
	if len(cues) == 0 {
		return plain(data)
	}
	return Transcript(cues), nil
}
