// Inline editing.
//
// Nothing on a canvas block is editable through a form on another page. If a
// person wants to fix a word, they fix it where the word is; if they want
// anything more than that, they ask the assistant, which is faster than any
// form and is the point of having one on the page.
//
// This is an enhancement, and it adds its own control rather than leaving a
// dead one behind: without JavaScript there is no Edit button, and the way
// to change something is to ask.
//
// A component says which of its elements carries which prop, with
// data-prop="title". This script swaps those elements for real, labelled
// form controls in place, so the semantics are native and the keyboard and
// screen reader behaviour is the browser's, not ours.
(function () {
  "use strict";

  var MULTILINE = { P: 1, DIV: 1, BLOCKQUOTE: 1 };

  function label(name) {
    return name.charAt(0).toUpperCase() + name.slice(1).replace(/[_-]/g, " ");
  }

  // field builds the control for one marked element, carrying the text that
  // is there now.
  function field(el, blockId) {
    // Structured text is edited as it is shown, by 11-prose-edit.js, when
    // that is here; otherwise as the Markdown it was written in.
    if (el.classList.contains("sw-prose") && el.hasAttribute("data-source") && window.swProseField) return window.swProseField(el, blockId);
    var name = el.getAttribute("data-prop");
    var id = "edit-" + blockId + "-" + name;
    if (el.getAttribute("data-kind") === "datetime") return dateField(el, id, name);
    var wrap = document.createElement("div");
    wrap.className = "sw-field sw-inline-field";

    var lab = document.createElement("label");
    lab.className = "sw-field__label";
    lab.setAttribute("for", id);
    lab.textContent = label(name);

    var source = el.hasAttribute("data-source") ? el.getAttribute("data-source") : null;
    var multiline = MULTILINE[el.tagName] === 1 || el.textContent.indexOf("\n") >= 0 || (source !== null && source.indexOf("\n") >= 0);
    var input = document.createElement(multiline ? "textarea" : "input");
    input.className = multiline ? "sw-field__textarea" : "sw-field__input";
    input.id = id;
    input.name = "prop-" + name;
    if (multiline) {
      var text = source !== null ? source : el.textContent;
      input.value = text;
    } else {
      input.value = el.textContent;
    }

    wrap.appendChild(lab);
    wrap.appendChild(input);

    // Save on submit, cancel on Escape.
    var form = document.createElement("form");
    form.className = "sw-inline-form sw-stack";
    form.method = "post";
    form.appendChild(wrap);

    var actions = document.createElement("div");
    actions.className = "sw-cluster";
    actions.innerHTML =
      '<button type="submit" class="sw-button sw-button--primary sw-pressable">Save</button>' +
      '<button type="button" class="sw-button sw-button--quiet sw-pressable" data-cancel>Cancel</button>';
    form.appendChild(actions);

    var block = el.closest("[data-block-id]");
    if (!block) return { wrap: wrap, input: input };
    block.insertBefore(form, block.firstChild);

    // Hide the read-only definition list so only the form is visible.
    var dl = block.querySelector("dl.sw-dl");
    if (dl) {
      dl.hidden = true;
      dl.style.display = "none";
    }

    for (var c = form.nextElementSibling; c; c = c.nextElementSibling) {
      if (c.classList.contains("sw-bar") || c.classList.contains("sw-visually-hidden")) continue;
      c.style.display = "none";
    }

    return { wrap: wrap, input: input };
  }

  // dateField builds a combined date+time picker for datetime props.
  function dateField(el, id, name) {
    var wrap = document.createElement("div");
    wrap.className = "sw-field sw-inline-field";

    var lab = document.createElement("label");
    lab.className = "sw-field__label";
    lab.setAttribute("for", id);
    lab.textContent = label(name);

    // Parse the current value into date and time parts.
    var val = el.textContent.trim();
    var day = "", time = "";
    if (val) {
      var parts = val.split(" ");
      day = parts[0] || "";
      time = parts.slice(1).join(" ") || "";
    }

    var hint = document.createElement("div");
    hint.className = "sw-visually-hidden";
    hint.textContent = "Format: YYYY-MM-DD HH:MM";

    var input = document.createElement("input");
    input.type = "datetime-local";
    input.className = "sw-field__input";
    input.id = id;
    input.name = "prop-" + name;
    if (day) {
      // Ensure the value is in the right format for datetime-local.
      var datePart = day.replace(/\//g, "-");
      if (time) {
        input.value = datePart + "T" + time.substring(0, 5);
      } else {
        input.value = datePart;
      }
    }

    wrap.appendChild(lab);
    wrap.appendChild(hint);
    wrap.appendChild(input);

    return { wrap: wrap, input: input };
  }

  // edit replaces the marked elements of one block with a small form.
  function edit(block) {
    if (block.querySelector(".sw-inline-form")) return;
    var marked = block.querySelectorAll("[data-prop]");
    if (!marked.length) return;
    var id = block.getAttribute("data-block-id");

    var form = document.createElement("form");
    form.className = "sw-inline-form sw-stack";
    form.method = "post";
    var action = block.getAttribute("data-edit-action");
    if (!action) { action = "/canvas/"+id+"/props"; }
    form.action = action;
    form.setAttribute("aria-label", "Edit this block");

    var first = null;
    for (var i = 0; i < marked.length; i++) {
      var built = field(marked[i], id);
      form.appendChild(built.wrap);
      if (!first) first = built.input;
    }

    var actions = document.createElement("div");
    actions.className = "sw-cluster";
    actions.innerHTML =
      '<button type="submit" class="sw-button sw-button--primary sw-pressable">Save</button>' +
      '<button type="button" class="sw-button sw-button--quiet sw-pressable" data-cancel>Cancel</button>';
    form.appendChild(actions);

    block.insertBefore(form, block.firstChild);

    // The fields stand in for the whole rendered block, not just the words
    // inside it, so an empty component shell is not left behind.
    var covered = [];

    // Hide the read-only definition list so only the form is visible.
    var dl = block.querySelector("dl.sw-dl");
    if (dl) {
      dl.hidden = true;
      dl.style.display = "none";
      covered.push(dl);
    }

    for (var c = form.nextElementSibling; c; c = c.nextElementSibling) {
      if (c.classList.contains("sw-bar") || c.classList.contains("sw-visually-hidden")) continue;
      c.style.display = "none";
      covered.push(c);
    }

    // Escape anywhere in the form, or the Cancel button, puts it back.
    form.addEventListener("keydown", function (e) {
      if (e.key === "Escape") { e.preventDefault(); cancel(block, form, covered); }
    });
    actions.querySelector("[data-cancel]").addEventListener("click", function () {
      cancel(block, form, covered);
    });

    if (first) first.focus();
  }

  function cancel(block, form, covered) {
    for (var i = 0; i < covered.length; i++) {
      covered[i].style.display = "";
      covered[i].hidden = false;
    }
    // Explicitly restore the definition list as well.
    var dl = block.querySelector("dl.sw-dl");
    if (dl) dl.hidden = false;
    form.remove();
    var btn = block.querySelector("[data-edit]");
    if (btn) btn.focus();
  }

  // Every block with editable text gets an Edit button, added here so it
  // never exists in a browser that cannot honour it.
  function arm(block) {
    if (!block.querySelector("[data-prop]")) return;
    var bar = block.querySelector(".sw-bar");
    if (!bar || bar.querySelector("[data-edit]")) return;
    var name = block.getAttribute("data-block-component") || "block";
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "sw-button sw-button--quiet sw-pressable";
    btn.setAttribute("data-edit", "");
    btn.innerHTML = 'Edit<span class="sw-visually-hidden"> ' + name + "</span>";
    btn.addEventListener("click", function () { edit(block); });
    bar.insertBefore(btn, bar.firstChild);
  }

  // Intercept Enter in the chat composer so pressing it submits instead of
  // inserting a newline. Shift+Enter is allowed through for newlines.
  function composeKeyHandler() {
    var textarea = document.querySelector("form.sw-compose textarea");
    if (!textarea) return;
    if (textarea._composeHandled) return;
    textarea._composeHandled = true;
    var form = textarea.closest("form.sw-compose");
    textarea.addEventListener("keydown", function (e) {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        form.submit();
      }
    });
  }

  // Intercept clicks on proposal accept/dismiss buttons so they POST via
  // fetch() instead of doing a full-page navigation, then remove the
  // answered proposal (and any others) from the page. Without JavaScript
  // the forms submit normally as plain HTML POSTs.
  function proposeHandler() {
    var btns = document.querySelectorAll(".sw-proposal [type=\"submit\"]");
    if (!btns || btns.length === 0) return;
    for (var i = 0; i < btns.length; i++) {
      (function (btn) {
        btn.addEventListener("click", function (e) {
          e.preventDefault();
          var form = btn.closest("form");
          fetch(form.action, { method: "POST" }).then(function () {
            document.querySelectorAll(".sw-proposal").forEach(function (p) { p.remove(); });
          });
        });
      })(btns[i]);
    }
  }

  function init() {
    document.querySelectorAll("[data-block-id]").forEach(arm);
    composeKeyHandler();
    proposeHandler();
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
