package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Several chats, and the chat's place on the page. A person can start a
// new chat, go back to an earlier one, or delete one, from a menu at the
// top of the chat. The chat is a block like any other, so the same menu
// lets a person move it to a pane or the middle and choose its width;
// the assistant can do the same with its tools. Popping it out opens
// /chat in a small window of its own.

// chatItem is one chat in the menu.
type chatItem struct {
	ID      string
	Title   string
	When    string
	Current bool
}

// chats lists every chat for the menu and names the current one.
func (s *Server) chats() (items []chatItem, current string) {
	id := s.app.Chat.Current()
	for _, c := range s.app.Chat.Conversations() {
		title := s.app.Chat.Title(c)
		item := chatItem{ID: c.ID, Title: title, When: c.CreatedAt.Local().Format("2 Jan"), Current: c.ID == id}
		if item.Current {
			current = title
		}
		items = append(items, item)
	}
	return items, current
}

func (s *Server) chatNew(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if _, err := s.app.Chat.NewChat(); err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, backTo(r.PostForm.Get("from")), http.StatusSeeOther)
}

func (s *Server) chatOpen(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if err := s.app.Chat.OpenChat(r.PostForm.Get("id")); err != nil {
		s.app.Chat.Notice("Chat not found")
	}
	http.Redirect(w, r, backTo(r.PostForm.Get("from")), http.StatusSeeOther)
}

func (s *Server) chatDelete(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if err := s.app.Chat.DeleteChat(r.PostForm.Get("id")); err != nil {
		s.app.Chat.Notice(err.Error())
	}
	http.Redirect(w, r, backTo(r.PostForm.Get("from")), http.StatusSeeOther)
}

// blockPlace moves a block to a region of the page, or changes its width,
// at the person's asking. The change is logged like one the assistant
// makes, so it can be undone.
func (s *Server) blockPlace(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(chat.BlockType, r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	r.ParseForm()
	fields := map[string]any{"actor": "human"}
	if region := r.PostForm.Get("region"); region != "" {
		fields["region"] = region
	}
	if span, err := strconv.Atoi(r.PostForm.Get("span")); err == nil && span >= 1 && span <= 12 {
		fields["span"] = span
	}
	name, _ := rec.Fields["component"].(string)
	if _, err := s.app.Store.Update(chat.BlockType, rec.ID, s.app.Chat.BlockFields(fields)); err != nil {
		s.app.Chat.Notice(err.Error())
	} else {
		s.recordPlace(rec, name, fields)
	}
	http.Redirect(w, r, backTo(r.PostForm.Get("from")), http.StatusSeeOther)
}

func (s *Server) recordPlace(rec *store.Record, name string, fields map[string]any) {
	detail := "moved"
	if region, ok := fields["region"].(string); ok {
		detail = "to the " + placeName(region)
	} else if span, ok := fields["span"].(int); ok {
		detail = fmt.Sprintf("%d of 12 columns wide", span)
	}
	chat.Record(s.app.Store, "human", chat.Change{Action: "updated", Component: name, ID: rec.ID, Detail: detail, Before: rec.Fields})
}

func placeName(region string) string {
	switch region {
	case "left":
		return "left pane"
	case "right":
		return "right pane"
	case "main":
		return "middle"
	}
	return region
}

// placeMenu is the Place menu for a block: where it sits and how wide it
// is, each a form, with the current choice marked.
func (s *Server) placeMenu(blk *store.Record, from string) template.HTML {
	region, _ := blk.Fields["region"].(string)
	if region == "" {
		region = "main"
	}
	span := 12
	switch v := blk.Fields["span"].(type) {
	case int:
		span = v
	case float64:
		span = int(v)
	case int64:
		span = int(v)
	}
	var b strings.Builder
	b.WriteString(`<details class="sw-chat__menu"><summary class="sw-button sw-button--quiet sw-pressable">Place</summary><div class="sw-chat__panel">`)
	b.WriteString(`<p class="sw-small sw-muted">Where</p><div class="sw-cluster">`)
	for _, r := range []string{"left", "main", "right"} {
		s.placeChoice(&b, blk.ID, from, "region", r, strings.ToUpper(placeName(r)[:1])+placeName(r)[1:], r == region)
	}
	b.WriteString(`</div>`)
	if region == "main" {
		b.WriteString(`<p class="sw-small sw-muted">Width</p><div class="sw-cluster">`)
		for _, w := range []struct {
			n     int
			label string
		}{{6, "Half"}, {9, "Wide"}, {12, "Full"}} {
			s.placeChoice(&b, blk.ID, from, "span", strconv.Itoa(w.n), w.label, w.n == span)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div></details>`)
	return template.HTML(b.String())
}

func (s *Server) placeChoice(b *strings.Builder, id, from, field, value, label string, current bool) {
	fmt.Fprintf(b, `<form method="post" action="/canvas/%s/place"><input type="hidden" name="from" value="%s"><input type="hidden" name="%s" value="%s"><button type="submit" class="sw-button sw-button--quiet sw-pressable"%s>%s</button></form>`,
		id, template.HTMLEscapeString(from), field, value, map[bool]string{true: ` aria-current="true"`, false: ""}[current], template.HTMLEscapeString(label))
}
