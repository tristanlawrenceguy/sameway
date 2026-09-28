// Icons for the prose editor's toolbar (12-prose-tools.js). Each button
// shows a familiar icon beside its word, never the icon alone: only a few
// icons are understood by nearly everyone, and a word next to one is what
// makes the rest quick to find and hard to mistake (NN/g "Icon Usability";
// W3C, "Use icons that help the user"). The icons are drawn in the text's
// own colour, at the text's size, hidden from screen readers, which hear
// the word.
(function () {
  "use strict";
  var PATHS = {
    heading: "M6 4v16M18 4v16M6 12h12",
    subheading: "M4 5v14M13 5v14M4 12h9M16.5 13.5a2 2 0 1 1 3.3 1.5L16.5 19h4",
    paragraph: "M13 4v16M17 4v16M19 4H9.5a4.5 4.5 0 0 0 0 9H13",
    bold: "M7 4h7a4 4 0 0 1 0 8H7zM7 12h8a4 4 0 0 1 0 8H7z",
    italic: "M19 4h-9M14 20H5M15 4 9 20",
    strikeThrough: "M16 5H10a3 3 0 0 0-2.8 4M14 12a4 4 0 0 1 0 8H7M4 12h16",
    code: "M16 18l6-6-6-6M8 6l-6 6 6 6",
    insertUnorderedList: "M9 6h12M9 12h12M9 18h12M4 6h.01M4 12h.01M4 18h.01",
    insertOrderedList: "M10 6h11M10 12h11M10 18h11M4 5l1-1v5M4 14.5a1.5 1.5 0 1 1 2.4 1.2L4 18h3",
    quote: "M5 17h3l2-4V7H4v6h3zM14 17h3l2-4V7h-6v6h3z",
    link: "M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7",
    more: "M5 12h.01M12 12h.01M19 12h.01",
    indent: "M3 8l4 4-4 4M21 12H11M21 6H11M21 18H11",
    outdent: "M7 8l-4 4 4 4M21 12H11M21 6H11M21 18H11",
    codeblock: "M3 5h18v14H3zM10 10l-2 2 2 2M14 10l2 2-2 2",
    image: "M3 4h18v16H3zM9 10.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3zM21 16l-5-5L5 20",
    table: "M3 4h18v16H3zM3 10h18M3 15h18M9 4v16M15 4v16",
    insertHorizontalRule: "M3 12h18"
  };
  // swProseLabel puts a command's icon and its word in a button.
  window.swProseLabel = function (b, cmd, word) {
    b.innerHTML = window.swProseIcon(cmd);
    var span = document.createElement("span");
    span.className = "sw-prose-tools__word";
    span.textContent = word;
    b.appendChild(span);
  };

  // swProseIcon is the icon for a command, as markup, or nothing.
  window.swProseIcon = function (cmd) {
    var d = PATHS[cmd];
    if (!d) return "";
    var heavy = cmd === "more" ? 3 : 2;
    return '<svg class="sw-prose-tools__icon" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="' +
      heavy + '" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="' + d + '"/></svg>';
  };
})();
