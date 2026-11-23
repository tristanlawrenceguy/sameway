package bench

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// languages are everyday requests in languages other than English: what
// Sameway keyed on English words (which tools a turn had, which rules,
// the days it read) is found out here. Each is checked by what was made,
// never by the words of the reply.
//
//	SAMEWAY_BENCH_SET=languages SAMEWAY_BENCH_MODEL=haiku go test ./internal/bench -run Assistant -v
func languages() []request {
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	twoDays := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
	return []request{
		{name: "es-task", say: "Añade una tarea para mañana: llamar al dentista",
			check: func(a *app.App, _ string) string {
				r := find(a, "task", "title", "dentista")
				return all(want(r != nil, "no dentist task"), want(r == nil || onDay(r, tomorrow), "it is not due tomorrow"))
			}},
		{name: "de-call", say: "Erinnere mich übermorgen um 15 Uhr daran, Oma anzurufen",
			check: func(a *app.App, _ string) string {
				r := titledAny(a, "oma")
				return all(want(r != nil, "nothing about Oma"), want(r == nil || onDay(r, twoDays), "it is not the day after tomorrow"),
					want(r == nil || atHour(r, 15), "it is not at 15:00"))
			}},
		{name: "fr-weekday", say: "Ajoute un rendez-vous chez le coiffeur jeudi prochain à 10h",
			check: func(a *app.App, _ string) string {
				r := titledAny(a, "coiffeur")
				return all(want(r != nil, "no hairdresser"), want(r == nil || onDay(r, next(time.Thursday)), "it is not on the coming Thursday"),
					want(r == nil || atHour(r, 10), "it is not at 10:00"))
			}},
		{name: "es-habit", say: "Quiero beber 8 vasos de agua al día",
			check: func(a *app.App, _ string) string {
				hs := list(a, "habit")
				var h *store.Record
				if len(hs) > 0 {
					h = hs[0]
				}
				return all(want(h != nil, "no habit"),
					want(h == nil || num(h.Fields["target"]) == 8 && has(h, "cadence", "day"), fmt.Sprintf("not 8 a day: %v", h)),
					block(a, "tracker", ""))
			}},
		{name: "de-delete", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "note", map[string]any{"title": "Alte Einkaufsliste", "body": "Milch, Eier."})
		}, say: "Lösche meine Notiz Alte Einkaufsliste",
			check: func(a *app.App, reply string) string {
				return want(find(a, "note", "title", "einkaufsliste") == nil, "the note is still there: "+reply)
			}},
		{name: "fr-week", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Renouveler le passeport", "due": tomorrow})
			mustCreate(t, a, "task", map[string]any{"title": "Envoyer la facture", "due": time.Now().AddDate(0, 0, 3).Format("2006-01-02")})
			mustCreate(t, a, "task", map[string]any{"title": "Préparer les vacances", "due": time.Now().AddDate(0, 0, 30).Format("2006-01-02")})
		}, say: "Qu'est-ce que j'ai à faire cette semaine ?",
			check: func(_ *app.App, reply string) string {
				r := strings.ToLower(reply)
				return all(want(strings.Contains(r, "passeport") && strings.Contains(r, "facture"), "the week's tasks are not in the answer"),
					want(!strings.Contains(r, "vacances"), "next month's task was given as this week's"))
			}},
		{name: "es-done", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Llamar al fontanero", "due": tomorrow})
		}, say: "Ya llamé al fontanero",
			check: func(a *app.App, _ string) string {
				r := find(a, "task", "title", "fontanero")
				return want(r != nil && r.Fields["done"] == true, "the plumber task is not ticked done")
			}},
		{name: "ja-birthday", say: "母の誕生日は3月3日です。毎年です。",
			check: func(a *app.App, reply string) string {
				r := titledAny(a, "誕生日")
				if r == nil {
					return "no birthday titled in Japanese; " + inJapanese(reply)
				}
				d := fmt.Sprint(r.Fields["starts"], r.Fields["at"], r.Fields["due"])
				return all(want(strings.Contains(d, "-03-03"), "it is not on 3 March: "+d),
					want(strings.Contains(strings.ToUpper(fmt.Sprint(r.Fields["repeat"])), "YEARLY"), "it does not repeat every year"), inJapanese(reply))
			}},
		{name: "ja-show", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "パスポートを更新する", "due": tomorrow})
		}, say: "タスクをページに表示して",
			check: func(a *app.App, reply string) string { return all(block(a, "collection", "task"), inJapanese(reply)) }},
	}
}

// inJapanese says whether a reply to a Japanese message is in Japanese:
// asked in Japanese, the assistant once answered and named what it made
// in English.
func inJapanese(reply string) string {
	for _, r := range reply {
		if r >= 0x3040 && r <= 0x30ff {
			return ""
		}
	}
	return "the reply is not in Japanese: " + reply
}

// titledAny is anyTitled for words that may be anywhere in a title.
func titledAny(a *app.App, words string) *store.Record { return anyTitled(a, words) }

// atHour says whether the record's day field holds that hour.
func atHour(r *store.Record, hour int) bool {
	for _, f := range []string{"at", "starts", "due"} {
		if v, _ := r.Fields[f].(string); v != "" && hourIs(r, f, hour) {
			return true
		}
	}
	return false
}
