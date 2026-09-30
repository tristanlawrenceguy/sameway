package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ArgsTrouble says what is wrong with a tool's arguments in the words of
// the one who sent them, never the decoder's: "position is a number; you
// sent a string", not "json: cannot unmarshal string into Go struct field
// .position of type int", which a model read as noise and sent again.
func ArgsTrouble(err error) string {
	var kind *json.UnmarshalTypeError
	var syntax *json.SyntaxError
	switch {
	case errors.As(err, &kind) && kind.Field != "" && sentWord(kind.Value) == "a number" && jsonKind(kind.Type) == "a number":
		return fmt.Sprintf("%s is a whole number; you sent %s. Send it again without a fraction.", kind.Field, strings.TrimPrefix(kind.Value, "number "))
	case errors.As(err, &kind) && kind.Field != "":
		return fmt.Sprintf("%s is %s; you sent %s. Send it again as %s.", kind.Field, jsonKind(kind.Type), sentWord(kind.Value), jsonKind(kind.Type))
	case errors.As(err, &kind):
		return fmt.Sprintf("the arguments are %s; send an object of them, such as {\"component\": \"text\", \"props\": {...}}", sentWord(kind.Value))
	case errors.As(err, &syntax):
		return fmt.Sprintf("the arguments are not valid JSON (the problem is at character %d)", syntax.Offset)
	}
	return "the arguments are not valid JSON"
}

// jsonKind is a Go type as JSON calls it.
func jsonKind(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "a string"
	case reflect.Bool:
		return "true or false"
	case reflect.Slice, reflect.Array:
		return "a list"
	case reflect.Map, reflect.Struct, reflect.Interface:
		return "an object"
	}
	return "a number"
}

// sentWord is what the decoder found, as JSON calls it: string, number,
// object, array, bool.
func sentWord(v string) string {
	switch {
	case v == "array":
		return "a list"
	case v == "bool":
		return "true or false"
	case v == "object":
		return "an object"
	case strings.HasPrefix(v, "number"):
		return "a number"
	}
	return "a " + v
}
