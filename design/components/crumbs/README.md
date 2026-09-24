# crumbs

The way back from where a person is. A record's page leads with the list it
belongs to; a deeper page leads with each place above it, outermost first.

Give `current` only when nothing else names the page: on a record's page the
heading below already says the record's name, and saying it twice makes a
screen reader say it twice. The separators are drawn by the stylesheet and are
never read out. Give a place its `dot` to show which of the person's lists it
is, the same colour the list has in the navigation.
