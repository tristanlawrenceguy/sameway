# image

A picture, sized to its column, with an optional caption under it.

`alt` is required, because every picture needs a decision about it. Say what
the picture shows that matters here, as you would to someone on the phone:
"The fern on the kitchen shelf, two new fronds". Leave it empty (`""`) only
when the picture is decoration and the words around it already say
everything; a screen reader then skips it.

A caption is seen by everyone and read after the picture. It does not stand
in for the alt text, which is for someone who cannot see the picture.

## Why it works this way

- **Its size is known before it loads**, so the words under it do not jump
  down as it arrives; a file's page reads it from the picture's own header
  ([web.dev on layout shift](https://web.dev/articles/optimize-cls#images-without-dimensions)).
- **Loaded at once when it is the first thing on the page** (`loading:
  eager`); lazy only lower down
  ([web.dev on lazy loading](https://web.dev/articles/browser-level-image-lazy-loading)).
- **A picture that moves starts still**, with Play and Stop beside it, so
  nothing moves on its own
  ([WCAG 2.2.2](https://www.w3.org/WAI/WCAG22/Understanding/pause-stop-hide.html)).
- **Never taller than most of a screen**, so the fields under it are in
  reach.
- **A picture that could not be loaded says so** in words beside it.
- **Alt text says what it shows, never "picture"**: a screen reader already
  says it is one ([WAI images tips](https://www.w3.org/WAI/tutorials/images/tips/)).
- **An uploaded file runs nothing when opened on its own**: an SVG or a web
  page can carry a script, and served from here it would act as the
  workspace, so files are served sandboxed and as the type they are.

Not done, and why: resized copies for each screen (one file per upload, and
scaling needs a dependency); a click-to-zoom view (Open the original already
shows it whole); descriptions written by the model (a person says what a
picture means here).
