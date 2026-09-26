# crumbs

The way back from where a person is. A record's page leads with the list it
belongs to; a deeper page leads with each place above it, outermost first.

Give `current` only when nothing else names the page: on a record's page the
heading below already says the record's name, and saying it twice makes a
screen reader say it twice, so a trail usually ends with the place above the
page, not the page. The chevrons between places are drawn with borders, not
characters, so nothing is read out for them. The landmark is named
Breadcrumb, the name screen reader users know; "You are here" would say they
are on the place it leads back to. Give a place its `dot` to show which of the person's lists it
is, the same colour the list has in the navigation.
