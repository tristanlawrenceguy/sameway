# pagination

The way through a long list a page at a time: Previous, the pages by number,
and Next. Every page is a plain link, so it works without scripts and each
page has its own address a person can keep or share.

Give it the address of the list (`href`), which may carry its own query such
as a search; the page goes on the end as `?page=2`, and the first page has
none. It shows the first and last pages and two either side of this one, with
a gap for the rest.

Say how many there are in all above the list, such as "230 notes"; this only
moves between them. Use it when a list could grow past what reads well on one
page, around 50 rows.
