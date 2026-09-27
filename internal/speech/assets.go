// Package speech turns a recording into its words on this computer, on
// whatever computer that is. The engine is sherpa-onnx, the one speech
// program published ready to run for Windows, macOS and Linux on both
// Intel and ARM; the model is Whisper base, which knows many languages.
// Neither is part of the sameway binary: both are downloaded once, when
// the person asks, from where their makers publish them, each checked
// against the fingerprint written here, and kept in the user's cache,
// shared by every workspace. The recording itself never leaves.
package speech

import "runtime"

// An asset is one file to fetch, and the fingerprint it must have.
type asset struct {
	URL    string
	SHA256 string
	Size   int64
	Name   string // what it is kept as
}

// EngineVersion is the sherpa-onnx release sameway runs.
const EngineVersion = "1.13.8"

const release = "https://github.com/k2-fsa/sherpa-onnx/releases/download/v" + EngineVersion + "/sherpa-onnx-v" + EngineVersion + "-"

// engines is the engine for each system, as its makers publish it: the
// shared, speech-to-text-only build, which on Windows needs no runtime
// installed beside it.
var engines = map[string]asset{
	"windows/amd64": {release + "win-x64-shared-MT-Release-no-tts.tar.bz2", "4b0a94f7b5c606b1b64a19a831c2127559e4b3d34e195465ebc7be73d9ed4783", 23271851, "engine.tar.bz2"},
	"windows/arm64": {release + "win-arm64-shared-MT-Release-no-tts.tar.bz2", "29a864324e658bef2a8b83bd3e12adae8b415a5a232d83902030e8c6efa37dbc", 21872501, "engine.tar.bz2"},
	"darwin/arm64":  {release + "osx-arm64-shared-no-tts.tar.bz2", "91b96512c4fa1960f8a9ed5360a6c8dda53a4b5015d0590244f14086a234557a", 18252168, "engine.tar.bz2"},
	"darwin/amd64":  {release + "osx-x64-shared-no-tts.tar.bz2", "03fd4cffd98b239d74b9253c270ff637661adb4d51a5a6c9e1f7486e48306db3", 20646120, "engine.tar.bz2"},
	"linux/amd64":   {release + "linux-x64-shared-no-tts.tar.bz2", "d0f96c8b65c6cd0974fada22737e337de81bc8cd2abbec2e39caf358b1eec5fc", 24802494, "engine.tar.bz2"},
	"linux/arm64":   {release + "linux-aarch64-shared-cpu.tar.bz2", "4e3734f82bc1379fd91f219f5869c7e9d03b7a4f7561907d8abca4849c51a789", 28091778, "engine.tar.bz2"},
}

// model is Whisper base in its compact form, from the revision pinned
// here, and the voice detector that cuts a recording into what was said.
const hf = "https://huggingface.co/csukuangfj/sherpa-onnx-whisper-base/resolve/bb53ee204431c90d314c1cc08d28d23e5b7927cc/"

var model = []asset{
	{hf + "base-encoder.int8.onnx", "0b8fb1304b6109976038efff5ace81720e00386f3ff6b54ee8c75291ca0a1e11", 29120534, "encoder.onnx"},
	{hf + "base-decoder.int8.onnx", "9759d217388a01b3a4c7c15533201067b48ae819c4daafc8624e64b9409dc02d", 130672026, "decoder.onnx"},
	{hf + "base-tokens.txt", "b34b360dbb493e781e479794586d661700670d65564001f23024971d1f2fa126", 816730, "tokens.txt"},
	{"https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/silero_vad.onnx", "9e2449e1087496d8d4caba907f23e0bd3f78d91fa552479bb9c23ac09cbb1fd6", 643854, "vad.onnx"},
}

// ModelName is what the model is called where a person reads it.
const ModelName = "Whisper base"

// system is this computer, as the engines are keyed.
func system() string { return runtime.GOOS + "/" + runtime.GOARCH }

// Supported says whether there is an engine for this computer.
func Supported() bool {
	_, ok := engines[system()]
	return ok
}

// DownloadSize is how much getting speech-to-text fetches, in bytes.
func DownloadSize() int64 {
	n := engines[system()].Size
	for _, a := range model {
		n += a.Size
	}
	return n
}
