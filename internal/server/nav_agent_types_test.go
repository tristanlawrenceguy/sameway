package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A kind of thing an agent made that holds nothing is left out of the
// person's navigation until something is in it; one the assistant made for
// the person is listed from the start.
func TestAnEmptyTypeAnAgentMadeIsNotInTheNavigation(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	made := postJSON(t, h, http.MethodPost, "/api/tools/add_type", map[string]any{"name": "test_adhoc", "description": "A test.", "properties": []map[string]any{{"name": "title", "kind": "string"}}})
	if made.Code >= 300 {
		t.Fatalf("an agent adds a type: %d %s", made.Code, made.Body)
	}
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_type", map[string]any{"name": "recipe", "description": "Recipes to cook.", "properties": []map[string]any{{"name": "title", "kind": "string"}}}),
		{Text: "Made recipes."},
	}}, nil
	postForm(t, h, "/chat", map[string][]string{"message": {"a place for recipes"}})

	nav := mainNav(t, get(t, h, "/").Body.String())
	if strings.Contains(nav, `href="/t/test_adhoc"`) {
		t.Errorf("an empty type an agent made is not in the navigation:\n%s", nav)
	}
	if !strings.Contains(nav, `href="/t/recipe"`) {
		t.Errorf("one the assistant made for the person is listed from the start:\n%s", nav)
	}
	postJSON(t, h, http.MethodPost, "/api/test_adhoc", map[string]any{"title": "One"})
	if nav := mainNav(t, get(t, h, "/").Body.String()); !strings.Contains(nav, `href="/t/test_adhoc"`) {
		t.Errorf("once it holds something, it is listed:\n%s", nav)
	}
}
