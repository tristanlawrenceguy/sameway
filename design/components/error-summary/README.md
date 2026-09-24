# error-summary

When a form is sent and some of its answers cannot be taken, say so at the top,
once, as a list: each problem in plain words with how to fix it, linking to the
field it is about. A person who sent a long form does not then have to hunt
for what went wrong, and a screen reader hears it as an alert when it arrives.

Name the field with `field`; the link goes to it, and with scripts pressing it
moves focus there and the field is marked invalid and described by its
problem. Use `href` for a problem that is somewhere else. Keep the same
message beside the field too, where the form shows one.
