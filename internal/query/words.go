package query

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Words says conditions the way a person would: done=false, due<today
// reads "not done and due before today", every one of them holding. A
// condition it cannot read is left as it was written.
func Words(t *schema.Type, where []string) string {
	var out []string
	for _, w := range where {
		if strings.TrimSpace(w) == "" {
			continue
		}
		c, err := Parse(t, w)
		if err != nil {
			out = append(out, w)
			continue
		}
		out = append(out, condWords(t, c))
	}
	if len(out) < 2 {
		return strings.Join(out, "")
	}
	return strings.Join(out[:len(out)-1], ", ") + " and " + out[len(out)-1]
}

func condWords(t *schema.Type, c Cond) string {
	name := strings.ToLower(strings.ReplaceAll(c.Field, "_", " "))
	f, has := t.Field(c.Field)
	if has && f.Label != "" {
		name = strings.ToLower(f.Label)
	}
	switch c.Field {
	case "created_at":
		name = "added"
	case "updated_at":
		name = "changed"
	case "id":
		if c.Op == "!=" {
			return "not this one"
		}
	}
	kind := kindOf(t, c.Field)
	if c.Value == "" {
		if c.Op == "!=" {
			return "with a " + name
		}
		return "no " + name
	}
	value := c.Value
	switch kind {
	case "bool":
		yes := (c.Value == "true") == (c.Op == "=")
		if yes {
			return name
		}
		return "not " + name
	case "enum":
		if has {
			value = f.ValueLabel(c.Value)
		}
	case "datetime":
		value = dateWords(c.Value)
		switch c.Op {
		case "<":
			return name + " before " + value
		case "<=":
			return name + " by " + value
		case ">":
			return name + " after " + value
		case ">=":
			return name + " from " + value
		}
	case "list":
		if c.Op == "=" {
			return name + " include " + value
		}
		if c.Op == "!=" {
			return name + " leave out " + value
		}
	}
	switch c.Op {
	case "=":
		return name + " is " + value
	case "!=":
		return name + " is not " + value
	case "~":
		return name + " has " + value
	case "<":
		return name + " under " + value
	case "<=":
		return name + " at most " + value
	case ">":
		return name + " over " + value
	case ">=":
		return name + " at least " + value
	}
	return c.String()
}

// dateWords says a date in a condition: today stays today, +7d is "7 days
// from now", and a written date is a day.
func dateWords(s string) string {
	switch s {
	case "today", "tomorrow", "yesterday", "now":
		return s
	}
	if len(s) > 2 && (s[0] == '+' || s[0] == '-') {
		n, unit := s[1:len(s)-1], s[len(s)-1:]
		units := map[string]string{"d": "day", "w": "week", "h": "hour", "m": "month", "y": "year"}
		if u, ok := units[unit]; ok {
			if n != "1" {
				u += "s"
			}
			if s[0] == '+' {
				return n + " " + u + " from now"
			}
			return n + " " + u + " ago"
		}
	}
	if d, err := time.Parse("2006-01-02", s); err == nil {
		return d.Format("Mon 2 Jan 2006")
	}
	return when.Text(s)
}
