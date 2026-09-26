# error-summary

When a form is sent and some of its answers cannot be taken, say so at the top,
once, as a list: each problem in plain words with how to fix it, linking to the
field it is about. A person who sent a long form does not then have to hunt
for what went wrong, and a screen reader hears it as an alert when it arrives.

Name the field with `field`; the link goes to it, and with scripts pressing it
moves focus there, opening the record's editor first when it is closed. Each
field it names is marked invalid, with the problem beside it in words, above
the input, as a field's own error is; a red edge alone would say it only in
colour. Focus goes to the summary once, when the problems arrive, so the list
is heard first and each link leads on. Use `href` for a problem that is
somewhere else; one with neither is plain words.

A page showing it has a window title that starts "Error:", the first thing a
screen reader says when the page arrives.
