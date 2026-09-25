// The toolbar of the prose editor (11-prose-edit.js): what a person can
// make of their words without knowing any syntax. Everything here is a
// browser editing command or a small piece of HTML dropped in; the server
// turns the result into Markdown on save, so this file only has to make
// the page look like what the person meant.
(function () {
  "use strict";

  // One strip, in groups: how a line reads, how words look, lists, a link.
  // What is used less waits behind More, so the strip stays one line.
  var GROUPS = [
    [["Heading", "heading"], ["Subheading", "subheading"], ["Text", "paragraph"]],
    [["B", "bold", "Bold"], ["I", "italic", "Italic"], ["S", "strikeThrough", "Strike"], ["Code", "code"]],
    [["List", "insertUnorderedList"], ["Numbered", "insertOrderedList"], ["Quote", "quote"]],
    [["Link", "link"]]
  ];
  var MORE = [["Indent", "indent"], ["Outdent", "outdent"], ["Code block", "codeblock"], ["Image", "image"], ["Table", "table"], ["Rule", "insertHorizontalRule"]];
  // Commands whose button shows whether the words at the caret already have it.
  var STATEFUL = { bold: 1, italic: 1, strikeThrough: 1, insertUnorderedList: 1, insertOrderedList: 1 };
  // Shortcuts, the ones most editors share, read from the key pressed, not
  // the character it types, so they work on every keyboard layout. None
  // use Ctrl+Alt: on many keyboards that is AltGr, which types characters.
  var KEYS = {
    heading: "Ctrl+Shift+1", subheading: "Ctrl+Shift+2", paragraph: "Ctrl+Shift+0",
    bold: "Ctrl+B", italic: "Ctrl+I", strikeThrough: "Ctrl+Shift+X", code: "Ctrl+E",
    insertUnorderedList: "Ctrl+Shift+8", insertOrderedList: "Ctrl+Shift+7", indent: "Tab", outdent: "Shift+Tab",
    quote: "Ctrl+Shift+9", link: "Ctrl+K"
  };
  // What a person types at the start of a line, then a space, to shape it.
  var TYPED = { "#": "heading", "##": "subheading", "-": "insertUnorderedList", "*": "insertUnorderedList", "1.": "insertOrderedList", ">": "quote" };
  var BLOCKS = /^(P|H[1-6]|BLOCKQUOTE|PRE|DIV)$/;

  // pressed says which command a key press asks for, if any.
  function pressed(e) {
    if (!(e.ctrlKey || e.metaKey) || e.altKey) return null;
    var c = e.code || "";
    var k = /^Key[A-Z]$/.test(c) ? c.slice(3) : /^Digit\d$/.test(c) ? c.slice(5) : (e.key.length === 1 ? e.key.toUpperCase() : e.key);
    var combo = "Ctrl+" + (e.shiftKey ? "Shift+" : "") + k;
    for (var cmd in KEYS) if (KEYS[cmd] === combo) return cmd;
    return null;
  }

  function escape(s) {
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }

  function range() {
    var sel = window.getSelection();
    return sel && sel.rangeCount ? sel.getRangeAt(0) : null;
  }

  // inside says whether the caret sits within an element of that tag.
  function inside(tag) {
    var r = range();
    for (var node = r ? r.startContainer : null; node; node = node.parentNode) {
      if (node.nodeType === 1 && node.tagName === tag) return node;
    }
    return null;
  }

  function blockOf(node, editor) {
    for (; node && node !== editor; node = node.parentNode) {
      if (node.nodeType === 1 && BLOCKS.test(node.tagName) && node.parentNode === editor) return node;
    }
    return null;
  }

  // setBlock makes a line a heading, text, a quote or code. With words
  // selected inside a paragraph, only those words become it, split out on
  // their own; otherwise the paragraph the caret is in does.
  function setBlock(editor, tag) {
    var r = range();
    var block = r && blockOf(r.startContainer, editor);
    if (!r || r.collapsed || !block || block !== blockOf(r.endContainer, editor)) {
      document.execCommand("formatBlock", false, tag);
      return;
    }
    var before = document.createRange(), after = document.createRange();
    before.setStart(block, 0); before.setEnd(r.startContainer, r.startOffset);
    after.setStart(r.endContainer, r.endOffset); after.setEnd(block, block.childNodes.length);
    function make(name, frag) { var el = document.createElement(name); el.appendChild(frag); return el; }
    var parts = [], head = before.cloneContents(), tail = after.cloneContents();
    if (head.textContent.trim()) parts.push(make(block.tagName, head));
    var picked = make(tag, r.cloneContents());
    parts.push(picked);
    if (tail.textContent.trim()) parts.push(make(block.tagName, tail));
    parts.forEach(function (p) { block.parentNode.insertBefore(p, block); });
    block.remove();
    var sel = window.getSelection(), at = document.createRange();
    at.selectNodeContents(picked);
    sel.removeAllRanges(); sel.addRange(at);
  }

  // tidy lifts a list or block the browser put inside a paragraph back out
  // of it, as the server reads it: a paragraph holds words, not a list.
  // The caret stays where it was.
  function tidy(editor) {
    var sel = window.getSelection(), node = sel.anchorNode, at = sel.anchorOffset;
    editor.querySelectorAll("p").forEach(function (p) {
      if (!p.querySelector(":scope > ul, :scope > ol, :scope > p, :scope > h1, :scope > h2, :scope > h3, :scope > h4, :scope > h5, :scope > h6, :scope > blockquote, :scope > pre, :scope > div")) return;
      while (p.firstChild) p.parentNode.insertBefore(p.firstChild, p);
      p.remove();
    });
    if (node && editor.contains(node)) {
      var r = document.createRange();
      r.setStart(node, Math.min(at, node.nodeType === 3 ? node.length : node.childNodes.length));
      r.collapse(true);
      sel.removeAllRanges(); sel.addRange(r);
    }
  }

  // run applies one command to the selection in the editor.
  function run(editor, cmd, level) {
    apply(editor, cmd, level);
    tidy(editor);
  }

  function apply(editor, cmd, level) {
    editor.focus();
    var r = range(), text = r ? r.toString() : "";
    switch (cmd) {
      case "heading": setBlock(editor, "H" + level); break;
      case "subheading": setBlock(editor, "H" + Math.min(6, level + 1)); break;
      case "paragraph": setBlock(editor, "P"); break;
      case "quote": setBlock(editor, inside("BLOCKQUOTE") ? "P" : "BLOCKQUOTE"); break;
      case "codeblock": setBlock(editor, inside("PRE") ? "P" : "PRE"); break;
      case "code":
        var code = inside("CODE");
        if (code) { code.replaceWith.apply(code, Array.prototype.slice.call(code.childNodes)); break; }
        if (r && !r.collapsed) {
          // The words keep what they had, bold or italic, inside the code.
          var c = document.createElement("code");
          c.appendChild(r.extractContents());
          r.insertNode(c);
        } else document.execCommand("insertHTML", false, "<code>code</code>");
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

  // typed turns "# " and the like at the start of a line into what they
  // mean, the way most editors do, and says whether it did.
  function typed(editor, level) {
    var r = range();
    if (!r || !r.collapsed) return false;
    var block = blockOf(r.startContainer, editor);
    if (!block) return false;
    var lead = document.createRange();
    lead.setStart(block, 0); lead.setEnd(r.startContainer, r.startOffset);
    var cmd = TYPED[lead.toString()];
    if (!cmd) return false;
    lead.deleteContents();
    // An emptied line would lose the caret to the line before, and the
    // format with it: the line keeps a break, and the caret stays in it.
    if (!block.textContent) block.innerHTML = "<br>";
    var here = document.createRange(), sel = window.getSelection();
    here.setStart(block, 0); here.collapse(true);
    sel.removeAllRanges(); sel.addRange(here);
    run(editor, cmd, level);
    return true;
  }

  // In a table, Tab moves to the next cell, and past the last it adds a
  // row. Shift+Tab from the first cell, and Tab from the last cell of an
  // empty row, go on as Tab does anywhere else, so nobody is kept there.
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
      var to = document.createRange();
      to.selectNodeContents(cells[at]);
      var sel = window.getSelection();
      sel.removeAllRanges();
      sel.addRange(to);
    });
  }

  // toolbar is one strip of buttons the arrow keys move between, as a
  // toolbar should, so Tab passes over it in one step.
  window.swProseToolbar = function (editor, level) {
    // A new line is a paragraph, as the server reads it, not a bare div.
    try { document.execCommand("defaultParagraphSeparator", false, "p"); } catch (e) { /* older browsers keep theirs */ }
    var bar = document.createElement("div");
    bar.className = "sw-prose-tools";
    bar.setAttribute("role", "toolbar");
    bar.setAttribute("aria-label", "Formatting");
    var buttons = [];
    function button(t, into) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "sw-button sw-button--quiet sw-pressable sw-prose-tools__" + t[1];
      b.textContent = t[0];
      var name = t[2] || t[0];
      if (t[2]) b.setAttribute("aria-label", name);
      if (KEYS[t[1]]) { b.title = name + " (" + KEYS[t[1]] + ")"; b.setAttribute("aria-keyshortcuts", KEYS[t[1]]); }
      b.tabIndex = buttons.length ? -1 : 0;
      if (STATEFUL[t[1]]) b.setAttribute("aria-pressed", "false");
      // A press must not take the selection away from the words it is about.
      b.addEventListener("mousedown", function (e) { e.preventDefault(); });
      b.addEventListener("click", function () { run(editor, t[1], level); reflect(); });
      into.appendChild(b);
      buttons.push([b, t[1]]);
    }
    GROUPS.forEach(function (g) {
      var group = document.createElement("div");
      group.className = "sw-prose-tools__group";
      g.forEach(function (t) { button(t, group); });
      bar.appendChild(group);
    });
    var extra = document.createElement("div");
    extra.className = "sw-prose-tools__group";
    extra.id = editor.id + "-more";
    extra.hidden = true;
    var more = document.createElement("button");
    more.type = "button";
    more.className = "sw-button sw-button--quiet sw-pressable";
    more.textContent = "More";
    more.tabIndex = -1;
    more.setAttribute("aria-expanded", "false");
    more.setAttribute("aria-controls", extra.id);
    more.addEventListener("mousedown", function (e) { e.preventDefault(); });
    more.addEventListener("click", function () {
      extra.hidden = !extra.hidden;
      more.setAttribute("aria-expanded", extra.hidden ? "false" : "true");
    });
    bar.appendChild(more);
    MORE.forEach(function (t) { button(t, extra); });
    bar.appendChild(extra);
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
      var items = Array.prototype.filter.call(bar.querySelectorAll("button"), function (b) { return !b.closest("[hidden]"); });
      var at = items.indexOf(document.activeElement);
      if (at < 0) return;
      e.preventDefault();
      var next = items[(at + (e.key === "ArrowRight" ? 1 : items.length - 1)) % items.length];
      bar.querySelectorAll("button").forEach(function (b) { b.tabIndex = -1; });
      next.tabIndex = 0;
      next.focus();
    });
    // The shortcuts above; Tab in a list nests it, and in a table moves on;
    // a space after "#", "-", "1." or ">" at the start of a line shapes it;
    // Enter on an empty quote or code line leaves it, as it leaves a list.
    editor.addEventListener("keydown", function (e) {
      var q = e.key === "Enter" && !e.shiftKey && (inside("BLOCKQUOTE") || inside("PRE"));
      if (q && !q.textContent.trim()) { e.preventDefault(); run(editor, "paragraph", level); return; }
      if (e.key === " " && !e.ctrlKey && !e.metaKey && typed(editor, level)) { e.preventDefault(); reflect(); return; }
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
