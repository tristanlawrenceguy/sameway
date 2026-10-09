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
	} {
		if got := Turn(c.mine, c.body); got != c.want {
			t.Errorf("%q from me=%v: %q, want %q", c.body, c.mine, got, c.want)
		}
	}
}
