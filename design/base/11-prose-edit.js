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
// to a textarea of the source. The toolbar itself is 12-prose-tools.js.
(function () {
  "use strict";

  function label(name) {
    return name.charAt(0).toUpperCase() + name.slice(1).replace(/[_-]/g, " ");
  }

  // convert asks the server for the other form of the same words.
  function convert(body, done) {
    fetch("/api/prose", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) })
      .then(function (r) { return r.json(); })
      .then(done);
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
    editor.setAttribute("aria-label", label(name));
    // tabindex="0" so keyboard Tab reaches the Body field.
    editor.setAttribute("tabindex", "0");
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
    source.setAttribute("aria-labelledby", lab.id);
    source.value = el.getAttribute("data-source") || "";
    source.hidden = true;

    var bar = window.swProseToolbar ? window.swProseToolbar(editor, level) : document.createElement("div");
    var toggle = document.createElement("button");
    toggle.type = "button";
    toggle.className = "sw-button sw-button--quiet sw-pressable";
    toggle.textContent = "Markdown";
    toggle.setAttribute("aria-pressed", "false");
    toggle.addEventListener("click", function () {
      if (source.hidden) {
        convert({ html: editor.innerHTML, level: level }, function (out) {
          source.value = out.markdown || source.value;
          editor.hidden = bar.hidden = true;
          html.disabled = true;
          source.hidden = false;
          toggle.hidden = false;
          toggle.textContent = "Rich text";
          toggle.setAttribute("aria-pressed", "true");
          source.focus();
        });
      } else {
        convert({ markdown: source.value, level: level }, function (out) {
          editor.innerHTML = out.html || editor.innerHTML;
          source.hidden = true;
          html.disabled = false;
          editor.hidden = bar.hidden = false;
          toggle.textContent = "Markdown";
          toggle.setAttribute("aria-pressed", "false");
          editor.focus();
        });
      }
    });
    var switcher = document.createElement("div");
    switcher.className = "sw-cluster sw-prose-switch";
    switcher.appendChild(toggle);

    wrap.appendChild(lab);
    wrap.appendChild(editor);
    wrap.appendChild(source);
    wrap.appendChild(switcher);
    wrap.appendChild(bar);
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
