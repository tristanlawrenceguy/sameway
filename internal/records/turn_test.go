package records

import "testing"

// Whose turn it is after a message: theirs when the person wrote it, the
// person's when it asks them something, nobody's when it answers or closes.
func TestWhoseTurnItIs(t *testing.T) {
	for _, c := range []struct {
		mine bool
		body string
		want string
	}{
		{true, "Here it is, attached.", "theirs"},
		{false, "Tuesday works. 3pm?", "yours"},
		{false, "One page is missing a signature. Please resend it.", "yours"},
		{false, "Thanks, got it!", ""},
		{false, "Booked, see you there.", ""},
		{false, "Let me know which you prefer.", "yours"},
		// Only the words written this time: not what is quoted, not a
		// link's address, not a signature.
		{false, "Thanks, got it!\n\nOn Mon 12 Oct 2026, Me <me@example.com> wrote:\n> Can you check it by Friday?", ""},
		{false, "Done.\n> Shall I book it?", ""},
		{false, "Booked: https://example.com/booking?id=42&utm=mail", ""},
		{false, "Here you are.\n\n-- \nJoe\nPlease consider the environment before printing.", ""},
		{false, "All sorted.\n\n-----Original Message-----\nFrom: Me\nCould you please call?", ""},
		{false, "See [the form](https://example.com/f?x=1). Can you sign it?", "yours"},
	} {
		if got := Turn(c.mine, c.body); got != c.want {
			t.Errorf("%q from me=%v: %q, want %q", c.body, c.mine, got, c.want)
		}
	}
}
