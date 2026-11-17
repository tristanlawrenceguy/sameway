package convert

import (
	"strings"
)

// The answer to an invitation comes back as a calendar of its own, sent
// as a reply (METHOD:REPLY, RFC 5546): Gmail, Outlook and Apple Mail all
// send one when someone presses Yes, No or Maybe. It names the event by
// its id and the one answering, with how they answered.

// Answer is one person's answer to an invitation.
type Answer struct {
	UID string
	// Who is who answered, as "Name <email>".
	Who string
	// State is accepted, declined or tentative.
	State string
}

// Answers reads the answers in a reply to an invitation, or none when the
// calendar is not a reply.
func Answers(data []byte) []Answer {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n ", "")
	text = strings.ReplaceAll(text, "\n\t", "")
	reply, uid := false, ""
	var out []Answer
	for _, line := range strings.Split(text, "\n") {
		name, params, value := property(line)
		switch name {
		case "METHOD":
			reply = strings.EqualFold(strings.TrimSpace(value), "REPLY")
		case "BEGIN":
			uid = ""
		case "UID":
			uid = strings.TrimSpace(value)
		case "ATTENDEE":
			state := strings.ToLower(params["PARTSTAT"])
			if who := attendee(params, value); who != "" && (state == "accepted" || state == "declined" || state == "tentative") {
				out = append(out, Answer{Who: who, State: state})
			}
		case "END":
			for i := range out {
				if out[i].UID == "" {
					out[i].UID = uid
				}
			}
		}
	}
	if !reply {
		return nil
	}
	kept := out[:0]
	for _, a := range out {
		if a.UID != "" {
			kept = append(kept, a)
		}
	}
	return kept
}
