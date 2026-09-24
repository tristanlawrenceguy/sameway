// The toolbar of the prose editor (11-prose-edit.js): what a person can
// make of their words without knowing any syntax. Everything here is a
// browser editing command or a small piece of HTML dropped in; the server
// turns the result into Markdown on save, so this file only has to make
// the page look like what the person meant.
(function () {
  "use strict";

  var TOOLS = [
    ["Heading", "heading"], ["Subheading", "subheading"], ["Text", "paragraph"],
    ["Bold", "bold"], ["Italic", "italic"], ["Strike", "strikeThrough"], ["Code", "code"],
    ["List", "insertUnorderedList"], ["Numbered", "insertOrderedList"], ["Indent", "indent"], ["Outdent", "outdent"],
    ["Quote", "quote"], ["Code block", "codeblock"], ["Link", "link"], ["Image", "image"], ["Table", "table"], ["Rule", "insertHorizontalRule"]
  ];
  // Commands whose button shows whether the words at the caret already have it.
  var STATEFUL = { bold: 1, italic: 1, strikeThrough: 1, insertUnorderedList: 1, insertOrderedList: 1 };
  // Shortcuts, the ones most editors share; Ctrl+B, Ctrl+I and Ctrl+Z are
  // the browser's own. Shown on each button so nobody has to guess.
  var KEYS = {
    heading: "Ctrl+Alt+1", subheading: "Ctrl+Alt+2", paragraph: "Ctrl+Alt+0",
    bold: "Ctrl+B", italic: "Ctrl+I", strikeThrough: "Ctrl+Shift+X", code: "Ctrl+E",
    insertUnorderedList: "Ctrl+Shift+8", insertOrderedList: "Ctrl+Shift+7", indent: "Tab", outdent: "Shift+Tab",
    quote: "Ctrl+Shift+9", codeblock: "Ctrl+Alt+C", link: "Ctrl+K"
  };

  // pressed says which command a key press asks for, if any.
  function pressed(e) {
    if (!(e.ctrlKey || e.metaKey)) return null;
    var k = e.key.length === 1 ? e.key.toUpperCase() : e.key;
    var combo = "Ctrl+" + (e.altKey ? "Alt+" : "") + (e.shiftKey ? "Shift+" : "") + k;
    if (e.shiftKey && e.code && e.code.indexOf("Digit") === 0) combo = "Ctrl+Shift+" + e.code.charAt(5);
    for (var cmd in KEYS) if (KEYS[cmd] === combo) return cmd;
    return null;
  }

  function escape(s) {
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }

  function selectedText() {
    var sel = window.getSelection();
    return sel && sel.rangeCount ? sel.toString() : "";
  }

  // inside says whether the caret sits within an element of that tag.
  function inside(tag) {
    var sel = window.getSelection();
    var node = sel && sel.rangeCount ? sel.getRangeAt(0).startContainer : null;
    for (; node; node = node.parentNode) {
      if (node.nodeType === 1 && node.tagName === tag) return node;
    }
    return null;
  }

  // run applies one command to the selection in the editor.
  function run(editor, cmd, level) {
    editor.focus();
    var text = selectedText();
    switch (cmd) {
      case "heading": document.execCommand("formatBlock", false, "H" + level); break;
      case "subheading": document.execCommand("formatBlock", false, "H" + Math.min(6, level + 1)); break;
      case "paragraph": document.execCommand("formatBlock", false, "P"); break;
      case "quote": document.execCommand("formatBlock", false, "BLOCKQUOTE"); break;
      case "codeblock":
        document.execCommand("formatBlock", false, inside("PRE") ? "P" : "PRE");
        break;
      case "code":
        var code = inside("CODE");
        if (code) { code.replaceWith.apply(code, Array.prototype.slice.call(code.childNodes)); break; }
        document.execCommand("insertHTML", false, "<code>" + escape(text || "code") + "</code>");
        break;
      case "link":
        if (inside("A")) { document.execCommand("unlink", false, null); break; }
        var href = window.prompt("Link to: a page here, like /t/note, or an address");
        if (!href) break;
        if (text) document.execCommand("createLink", false, href);
        else document.execCommand("insertHTML", false, '<a href="' + escape(href) + '">' + escape(href) + "</a>");
        break;
      case "image":
        var src = window.prompt("Picture address: a file here, like /files/<id>, or an address");
        if (!src) break;
        var alt = window.prompt("Describe the picture for someone who cannot see it") || "";
        // Asked twice: empty the second time is decoration, meant.
        if (!alt) alt = window.prompt("Without a description, someone who cannot see it hears nothing. What does it show? Leave empty only if it is decoration.") || "";
        document.execCommand("insertHTML", false, '<img src="' + escape(src) + '" alt="' + escape(alt) + '">');
        break;
      case "table":
        document.execCommand("insertHTML", false,
          "<table><thead><tr><th>Heading</th><th>Heading</th></tr></thead><tbody><tr><td>Cell</td><td>Cell</td></tr></tbody></table><p><br></p>");
        break;
      default: document.execCommand(cmd, false, null);
    }
  }

  // In a table, Tab moves to the next cell, and past the last cell it adds
  // a row, so a table grows without a button. Tab never keeps a person in
  // the table: Shift+Tab from the first cell, and Tab from the last cell
  // of a row still empty, go on as Tab does anywhere else.
  function tableKeys(editor) {
    editor.addEventListener("keydown", function (e) {
      if (e.key !== "Tab") return;
      var cell = inside("TD") || inside("TH");
      if (!cell) return;
      var cells = cell.closest("table").querySelectorAll("th, td");
      var at = Array.prototype.indexOf.call(cells, cell) + (e.shiftKey ? -1 : 1);
      if (at < 0) return;
      if (at >= cells.length && !cell.closest("tr").textContent.trim()) return;
      e.preventDefault();
      if (at >= cells.length) {
        var row = cell.closest("tr").cloneNode(true);
        row.querySelectorAll("th, td").forEach(function (c) { c.innerHTML = "<br>"; });
        var body = cell.closest("table").tBodies[0] || cell.closest("table");
        body.appendChild(row);
        cells = cell.closest("table").querySelectorAll("th, td");
      }
      var range = document.createRange();
      range.selectNodeContents(cells[at]);
      var sel = window.getSelection();
      sel.removeAllRanges();
      sel.addRange(range);
    });
  }

  // toolbar is a row of buttons the arrow keys move between, as a toolbar
  // should, so Tab passes over it in one step.
  window.swProseToolbar = function (editor, level) {
    var bar = document.createElement("div");
    bar.className = "sw-cluster sw-prose-tools";
    bar.setAttribute("role", "toolbar");
    bar.setAttribute("aria-label", "Formatting");
    var buttons = [];
    TOOLS.forEach(function (t, i) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "sw-button sw-button--quiet sw-pressable";
      b.textContent = t[0];
      if (KEYS[t[1]]) { b.title = t[0] + " (" + KEYS[t[1]] + ")"; b.setAttribute("aria-keyshortcuts", KEYS[t[1]]); }
      // One stop for the whole toolbar, after the words it formats: Tab
      // reaches the first button, the arrow keys the rest.
      b.tabIndex = i === 0 ? 0 : -1;
      if (STATEFUL[t[1]]) b.setAttribute("aria-pressed", "false");
      // A press must not take the selection away from the words it is about.
      b.addEventListener("mousedown", function (e) { e.preventDefault(); });
      b.addEventListener("click", function () { run(editor, t[1], level); reflect(); });
      bar.appendChild(b);
      buttons.push([b, t[1]]);
    });
    // reflect shows on each button whether the words at the caret have it.
    function reflect() {
      buttons.forEach(function (p) {
        if (STATEFUL[p[1]]) p[0].setAttribute("aria-pressed", document.queryCommandState(p[1]) ? "true" : "false");
        if (p[1] === "link") p[0].textContent = inside("A") ? "Unlink" : "Link";
      });
    }
    document.addEventListener("selectionchange", function () { if (editor.contains(document.activeElement)) reflect(); });
    bar.addEventListener("keydown", function (e) {
      if (e.key !== "ArrowRight" && e.key !== "ArrowLeft") return;
      var items = bar.querySelectorAll("button");
      var at = Array.prototype.indexOf.call(items, document.activeElement);
      if (at < 0) return;
      e.preventDefault();
      var next = items[(at + (e.key === "ArrowRight" ? 1 : items.length - 1)) % items.length];
      items[at].tabIndex = -1;
      next.tabIndex = 0;
      next.focus();
    });
    // The shortcuts above; Tab in a list nests it, and in a table moves on.
    editor.addEventListener("keydown", function (e) {
      var cmd = pressed(e);
      if (!cmd && e.key === "Tab" && !inside("TD") && !inside("TH") && inside("LI")) cmd = e.shiftKey ? "outdent" : "indent";
      if (!cmd || cmd === "bold" || cmd === "italic") return;
      e.preventDefault();
      run(editor, cmd, level);
      reflect();
    });
    tableKeys(editor);
    return bar;
  };
})();
