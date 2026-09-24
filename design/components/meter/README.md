# meter

How far along a known range something is, as a thin bar: 3 of 8 glasses, 40
of 60 hours, a budget part spent. Give it a label that names the thing and
words (`text`) that say the amount the way a person would, such as "3 of 8
glasses"; a screen reader reads the words, not a bare number.

`state` colours it: `going` in the list colour, `met` in success green, `over`
in the warning colour for a limit gone past. The words should say it too,
because colour alone does not.

A value past `max` fills the bar and is read as full; say how far over in the
words. For work in progress with no known end, use a status instead.
