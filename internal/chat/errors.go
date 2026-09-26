package chat

import "regexp"

// humanizeValidationError replaces JSON Schema validation error syntax with
// plain-language text so the model never repeats raw schema internals in chat.
func humanizeValidationError(err string) string {
	// Replace raw JSON Schema phrases but keep field names and enum choices
	// visible so the model can learn from them.
	err = regexp.MustCompile(`(?i)additional properties .* not allowed`).ReplaceAllString(err, "something I don't recognise")
	return err
}
