# alert

Use an alert for a short message that changes what the person does next: a
save confirmation, a validation problem, a failed request. The kind is spoken
as a word ("Error:") as well as shown as a colour. Keep the message to one or
two sentences and say how to fix the problem when there is one.

Provide an icon prop (a unicode character like ⚠ or ℹ) so that the alert's
meaning reaches users who cannot perceive colour — this satisfies WCAG 1.4.1
(non-colour visual presentation). The kind text prefix ("Error:", "Note:")
and the icon together convey meaning independently of CSS styling.
