// Editing words where they are shown.
//
// Structured text is rendered on the page. Editing it means changing that
// rendering, not a box of syntax beside it: a person makes a heading, a
// list or a link with a button or a shortcut, and sees it as it will be.
// The server turns the result back into Markdown when it is saved
// (html-<name>, with level-<name> saying what level a # line was shown at).
//
// The raw Markdown stays one button away until the editor does everything
// a person needs; then that button can go. 08-edit.js hands any element
// with data-source to swProseField, and without this file it falls back
// to a textarea of the source.
(function () {
  "use strict";

  var TOOLS = [
    ["Heading", "heading"], ["Subheading", "subheading"], ["Text", "paragraph"],
    ["Bold", "bold"], ["Italic", "italic"], ["List", "insertUnorderedList"],
    ["Numbered", "insertOrderedList"], ["Quote", "quote"], ["Link", "link"]
  ];

  function label(name) {
    return name.charAt(0).toUpperCase() + name.slice(1).replace(/[_-]/g, " ");
  }

  // run applies one formatting command to the selection in the editor.
  function run(editor, cmd, level) {
    editor.focus();
    switch (cmd) {
      case "heading": document.execCommand("formatBlock", false, "H" + level); break;
      case "subheading": document.execCommand("formatBlock", false, "H" + Math.min(6, level + 1)); break;
      case "paragraph": document.execCommand("formatBlock", false, "P"); break;
      case "quote": document.execCommand("formatBlock", false, "BLOCKQUOTE"); break;
      case "link":
        var href = window.prompt("Link to: a page here, like /t/note, or an address");
        if (href) document.execCommand("createLink", false, href);
        break;
      default: document.execCommand(cmd, false, null);
    }
  }

  // convert asks the server for the other form of the same words.
  function convert(body, done) {
    fetch("/api/prose", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) })
      .then(function (r) { return r.json(); })
      .then(done);
  }

  // toolbar is a row of buttons the arrow keys move between, as a toolbar
  // should, so Tab passes over it in one step.
  function toolbar(editor, level) {
    var bar = document.createElement("div");
    bar.className = "sw-cluster sw-prose-tools";
    bar.setAttribute("role", "toolbar");
    bar.setAttribute("aria-label", "Formatting");
    TOOLS.forEach(function (t, i) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "sw-button sw-button--quiet sw-pressable";
      b.textContent = t[0];
      b.tabIndex = i === 0 ? 0 : -1;
      // A press must not take the selection away from the words it is about.
      b.addEventListener("mousedown", function (e) { e.preventDefault(); });
      b.addEventListener("click", function () { run(editor, t[1], level); });
      bar.appendChild(b);
    });
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
    return bar;
  }

  // swProseField builds the editor for one rendered field: the prose
  // itself, editable, with its toolbar, and the Markdown behind a button.
  window.swProseField = function (el, blockId) {
    var name = el.getAttribute("data-prop");
    var level = parseInt(el.getAttribute("data-prose-level"), 10) || 2;
    var id = "edit-" + blockId + "-" + name;
    var wrap = document.createElement("div");
    wrap.className = "sw-field sw-inline-field sw-prose-field";

    var lab = document.createElement("span");
    lab.className = "sw-field__label";
    lab.id = id + "-label";
    lab.textContent = label(name);

    var editor = document.createElement("div");
    editor.className = "sw-prose sw-field__textarea sw-prose-editor";
    editor.id = id;
    editor.contentEditable = "true";
    editor.setAttribute("role", "textbox");
    editor.setAttribute("aria-multiline", "true");
    editor.setAttribute("aria-labelledby", lab.id);
    editor.innerHTML = el.innerHTML;

    var html = document.createElement("input");
    html.type = "hidden";
    html.name = "html-" + name;
    var lvl = document.createElement("input");
    lvl.type = "hidden";
    lvl.name = "level-" + name;
    lvl.value = String(level);

    var source = document.createElement("textarea");
    source.className = "sw-field__textarea";
    source.name = "prop-" + name;
    source.rows = 8;
    source.hidden = true;
    source.disabled = true;
    source.setAttribute("aria-labelledby", lab.id);
    source.value = el.getAttribute("data-source") || "";

    var bar = toolbar(editor, level);
    var toggle = document.createElement("button");
    toggle.type = "button";
    toggle.className = "sw-button sw-button--quiet sw-pressable";
    toggle.textContent = "Markdown";
    toggle.setAttribute("aria-pressed", "false");
    toggle.addEventListener("click", function () {
      if (source.disabled) {
        convert({ html: editor.innerHTML, level: level }, function (out) {
          source.value = out.markdown || source.value;
          editor.hidden = bar.hidden = true;
          html.disabled = true;
          source.hidden = source.disabled = false;
          toggle.textContent = "Rich text";
          toggle.setAttribute("aria-pressed", "true");
          source.focus();
        });
      } else {
        convert({ markdown: source.value, level: level }, function (out) {
          editor.innerHTML = out.html || editor.innerHTML;
          source.hidden = source.disabled = true;
          html.disabled = false;
          editor.hidden = bar.hidden = false;
          toggle.textContent = "Markdown";
          toggle.setAttribute("aria-pressed", "false");
          editor.focus();
        });
      }
    });
    bar.appendChild(toggle);

    wrap.appendChild(lab);
    wrap.appendChild(bar);
    wrap.appendChild(editor);
    wrap.appendChild(source);
    wrap.appendChild(html);
    wrap.appendChild(lvl);
    return { wrap: wrap, input: editor };
  };

  // What the editor holds goes with the form when it is sent.
  document.addEventListener("submit", function (e) {
    e.target.querySelectorAll(".sw-prose-field").forEach(function (f) {
      var editor = f.querySelector(".sw-prose-editor");
      var html = f.querySelector("input[name^=html-]");
      if (editor && html && !html.disabled) html.value = editor.innerHTML;
    });
  }, true);
})();
