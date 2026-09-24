# when-field

Asks when something is, the way a person says it: 19 Sep, next Friday,
tomorrow 2pm. People know the day they mean and type it faster than they find
it in a month; the platform's own picker stands beside the words for anyone
who would rather look, and picking a day writes it into the words, keeping any
time already typed.

The words are what is sent and the server reads them. The picker sends nothing
and is hidden until scripts run, since without them it could not write into
the words. Pass `day` as YYYY-MM-DD so the picker opens on the day already
there. The hint is always shown: it is the example of what can be typed.

Use the datepicker instead when a day must be chosen from a month and never
typed.
