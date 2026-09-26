package render

import "fmt"

// A table's cells, as the template draws them. Each row is as long as the
// columns, so no cell sits under the wrong heading or under none: a short
// row is filled out with empty cells, and what is past the last column is
// left out. The first cell names its row, as a row header, unless the
// table says otherwise; a column named as numbers is set to the right, so
// its figures line up to be compared; an empty cell says so.

type tableCell struct {
	Text  string
	Head  bool
	Num   bool
	Empty bool
}

// tableCells is rows laid out under columns: rowHeader is true unless
// false is given, numbers names the columns whose values are numbers.
func tableCells(rows, columns, numbers, rowHeader any) [][]tableCell {
	cols := stringsOf(columns)
	nums := map[string]bool{}
	for _, n := range stringsOf(numbers) {
		nums[n] = true
	}
	head := fmt.Sprint(rowHeader) != "false"
	var out [][]tableCell
	list, _ := rows.([]any)
	for _, r := range list {
		cells := stringsOf(r)
		row := make([]tableCell, len(cols))
		for i := range cols {
			c := tableCell{Head: head && i == 0, Num: nums[cols[i]]}
			if i < len(cells) {
				c.Text = cells[i]
			}
			c.Empty = c.Text == ""
			row[i] = c
		}
		out = append(out, row)
	}
	return out
}

// isNumbers is whether the column is one whose values are numbers.
func isNumbers(column string, numbers any) bool {
	for _, n := range stringsOf(numbers) {
		if n == column {
			return true
		}
	}
	return false
}

func stringsOf(v any) []string {
	switch l := v.(type) {
	case []string:
		return l
	case []any:
		out := make([]string, 0, len(l))
		for _, x := range l {
			if x == nil {
				out = append(out, "")
				continue
			}
			out = append(out, fmt.Sprint(x))
		}
		return out
	}
	return nil
}
