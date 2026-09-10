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
    var name = el.getAttribute("data-prop");
    var id = "edit-" + blockId + "-" + name;
    var wrap = document.createElement("div");
    wrap.className = "sw-field sw-inline-field";

    var lab = document.createElement("label");
    lab.className = "sw-field__label";
    lab.setAttribute("for", id);
    lab.textContent = label(name);

    var multiline = MULTILINE[el.tagName] === 1 || el.textContent.indexOf("\n") >= 0;
    var input = document.createElement(multiline ? "textarea" : "input");
    input.className = multiline ? "sw-field__textarea" : "sw-field__input";
    input.id = id;
    input.name = "prop-" + name;
    if (multiline) {
      input.rows = Math.min(10, Math.max(3, el.textContent.split("\n").length + 1));
      input.value = el.innerText.replace(/\n{3,}/g, "\n\n").trim();
    } else {
      input.type = "text";
      input.value = el.textContent.trim();
    }
    wrap.appendChild(lab);
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
    form.action = "/canvas/" + id + "/props";
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
    for (var c = form.nextElementSibling; c; c = c.nextElementSibling) {
      if (c.classList.contains("sw-bar") || c.classList.contains("sw-visually-hidden")) continue;
      c.hidden = true;
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
    for (var i = 0; i < covered.length; i++) covered[i].hidden = false;
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

  function init() {
    document.querySelectorAll("[data-block-id]").forEach(arm);
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
