package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// setFlags collects repeated --set key=value pairs.
type setFlags []string

func (s *setFlags) String() string     { return strings.Join(*s, ",") }
func (s *setFlags) Set(v string) error { *s = append(*s, v); return nil }

// contentCmd handles `sameway <type> <verb> ...` for every content type.
func (c *ctx) contentCmd(typeName string) error {
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	t, ok := a.Types.Get(typeName)
	if !ok {
		return fmt.Errorf("unknown command or content type %q (types: %s). Run `sameway help`", typeName, strings.Join(a.Types.Names(), ", "))
	}
	if len(c.args) == 0 {
		return fmt.Errorf("usage: sameway %s list|get|create|update|delete", t.Name)
	}
	verb, rest := c.args[0], c.args[1:]
	fs := flag.NewFlagSet(t.Name+" "+verb, flag.ContinueOnError)
	var sets setFlags
	fs.Var(&sets, "set", "field=value (repeatable)")
	data := fs.String("data", "", "JSON object of fields")
	order := fs.String("order", "", "field to order by")
	limit := fs.Int("limit", 0, "max records")
	positional, err := parseMixed(fs, rest)
	if err != nil {
		return err
	}
	switch verb {
	case "list":
		recs, err := a.Store.List(t.Name, store.ListOptions{OrderBy: *order, Limit: *limit})
		if err != nil {
			return err
		}
		c.print(recs, func() {
			if len(recs) == 0 {
				fmt.Fprintf(c.Stdout, "no %s records\n", t.Name)
			}
			for _, r := range recs {
				fmt.Fprintf(c.Stdout, "%s  %s\n", r.ID, summary(r, t.Title))
			}
		})
	case "get":
		if len(positional) != 1 {
			return fmt.Errorf("usage: sameway %s get <id>", t.Name)
		}
		rec, err := a.Store.Get(t.Name, positional[0])
		if err != nil {
			return err
		}
		c.print(rec, func() { printRecord(c, rec) })
	case "create":
		fields, err := fieldsFrom(sets, *data)
		if err != nil {
			return err
		}
		rec, err := a.Store.Create(t.Name, fields)
		if err != nil {
			return err
		}
		c.print(rec, func() { fmt.Fprintf(c.Stdout, "created %s %s\n", t.Name, rec.ID) })
	case "update":
		if len(positional) != 1 {
			return fmt.Errorf("usage: sameway %s update <id> --set field=value", t.Name)
		}
		fields, err := fieldsFrom(sets, *data)
		if err != nil {
			return err
		}
		rec, err := a.Store.Update(t.Name, positional[0], fields)
		if err != nil {
			return err
		}
		c.print(rec, func() { fmt.Fprintf(c.Stdout, "updated %s %s\n", t.Name, rec.ID) })
	case "delete":
		if len(positional) != 1 {
			return fmt.Errorf("usage: sameway %s delete <id>", t.Name)
		}
		if err := a.Store.Delete(t.Name, positional[0]); err != nil {
			return err
		}
		c.print(map[string]any{"deleted": positional[0]}, func() { fmt.Fprintf(c.Stdout, "deleted %s %s\n", t.Name, positional[0]) })
	default:
		return fmt.Errorf("unknown verb %q for %s (use list, get, create, update, delete)", verb, t.Name)
	}
	return nil
}

func fieldsFrom(sets []string, data string) (map[string]any, error) {
	fields := map[string]any{}
	if data != "" {
		if err := json.Unmarshal([]byte(data), &fields); err != nil {
			return nil, errors.New("--data must be a JSON object: " + err.Error())
		}
	}
	for _, kv := range sets {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("--set needs field=value, got %q", kv)
		}
		fields[strings.TrimSpace(k)] = v
	}
	if len(fields) == 0 {
		return nil, errors.New("nothing to save: pass --set field=value or --data '{...}'")
	}
	return fields, nil
}

func summary(r *store.Record, title string) string {
	if title != "" {
		if s, ok := r.Fields[title].(string); ok {
			return strings.SplitN(s, "\n", 2)[0]
		}
	}
	b, _ := json.Marshal(r.Fields)
	return string(b)
}

func printRecord(c *ctx, r *store.Record) {
	fmt.Fprintf(c.Stdout, "id: %s\ncreated: %s\nupdated: %s\n", r.ID, r.CreatedAt.Format("2006-01-02 15:04:05"), r.UpdatedAt.Format("2006-01-02 15:04:05"))
	for k, v := range r.Fields {
		fmt.Fprintf(c.Stdout, "%s: %v\n", k, v)
	}
}
