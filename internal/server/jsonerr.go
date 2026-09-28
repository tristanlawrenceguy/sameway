package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

// jsonTrouble says what is wrong with a JSON body in the caller's terms,
// never the decoder's: no Go type names, no "invalid character" puzzles.
func jsonTrouble(err error) string {
	var syntax *json.SyntaxError
	var kind *json.UnmarshalTypeError
	switch {
	case errors.As(err, &syntax):
		return fmt.Sprintf("it is not valid JSON (the problem is at character %d)", syntax.Offset)
	case errors.As(err, &kind) && kind.Field != "":
		return fmt.Sprintf("%s should be %s, not %s", kind.Field, kindWord(kind.Type), kind.Value)
	case errors.As(err, &kind):
		return fmt.Sprintf("it is %s, not an object", withArticle(kind.Value))
	case errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, io.EOF):
		return "it ends before the JSON is complete"
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		return strings.TrimPrefix(err.Error(), "json: unknown field ") + " is not one of them"
	}
	return "it is not valid JSON"
}

func kindWord(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "text"
	case reflect.Bool:
		return "true or false"
	case reflect.Slice, reflect.Array:
		return "a list"
	case reflect.Map, reflect.Struct, reflect.Interface:
		return "an object"
	}
	return "a number"
}

func withArticle(value string) string {
	if strings.ContainsRune("aeiou", rune(value[0])) {
		return "an " + value
	}
	return "a " + value
}
