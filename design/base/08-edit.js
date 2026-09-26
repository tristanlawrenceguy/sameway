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

  // control is a copy of one of the design system's own controls, which
  // the page carries in a template it does not show (editfields.go), with
  // this field's label, name and id put in. Every field the editor makes
  // is the component itself: the same markup, classes and wiring a page
  // gets when the server renders it.
  function control(kind, el, id, name) {
    var t = document.getElementById("sw-controls");
    var held = t && t.content.querySelector('[data-control="' + kind + '"]');
    if (!held) return null;
    var wrap = held.firstElementChild.cloneNode(true);
    wrap.classList.add("sw-inline-field");
    var input = wrap.querySelector("input, select, textarea");
    var lab = wrap.querySelector("label");
    lab.textContent = el.getAttribute("data-label") || label(name);
    lab.setAttribute("for", id);
    input.id = id;
    input.name = "prop-" + name;
    var hint = wrap.querySelector("[id$='-hint']");
    if (hint) {
      hint.id = id + "-hint";
      input.setAttribute("aria-describedby", hint.id);
    }
    return { wrap: wrap, input: input };
  }

  // field makes the control for one marked element, carrying what is
  // there now (for example: Body on a note or task detail page). Which
  // control it is comes from the mark: a yes or no, a day, a choice, a
  // number, a paragraph or a line.
  function field(el, blockId) {
    // Structured text is edited as it is shown, by 11-prose-edit.js, when
    // that is here; otherwise as the Markdown it was written in.
    if (el.classList.contains("sw-prose") && el.hasAttribute("data-source") && window.swProseField) return window.swProseField(el, blockId);
    var name = el.getAttribute("data-prop");
    var id = "edit-" + blockId + "-" + name;
    var kind = el.getAttribute("data-kind");
    var source = el.hasAttribute("data-source") ? el.getAttribute("data-source") : null;
    var built;
    if (kind === "bool") {
      built = control("checkbox", el, id, name);
      built.input.checked = source === "true";
      // An unticked box sends nothing, so a hidden no stands behind it, the
      // way the mark component does it; the box, first, wins when ticked.
      var no = document.createElement("input");
      no.type = "hidden";
      no.name = "prop-" + name;
      no.value = "false";
      built.wrap.appendChild(no);
      return built;
    }
    if (kind === "datetime") {
      // The when-field component: words, and its picker, which its own
      // enhance.js shows and wires.
      built = control("when", el, id, name);
      built.input.value = el.textContent.trim();
      var pick = built.wrap.querySelector(".sw-when-field__pick");
      pick.setAttribute("aria-label", "Pick a day for " + (el.getAttribute("data-label") || label(name)));
      if (/^\d{4}-\d{2}-\d{2}/.test(source || "")) pick.value = source.slice(0, 10);
      if (window.swWhenField) window.swWhenField(built.wrap);
      return built;
    }
    if (el.hasAttribute("data-options")) return choose(el, id, name, source !== null ? source : el.textContent.trim());
    if (kind === "lookup") {
      // The lookup component: the box shows the name, a hidden field sends
      // the id, and its enhance.js offers matches as a person types.
      built = control("lookup", el, id, name);
      var box = built.input, chosen = built.wrap.querySelector("input[type=hidden]");
      box.removeAttribute("name");
      chosen.name = "prop-" + name;
      chosen.value = source || "";
      box.value = el.getAttribute("data-title") || "";
      built.wrap.setAttribute("data-to", el.getAttribute("data-to"));
      built.wrap.querySelector("datalist").id = id + "-list";
      box.setAttribute("list", id + "-list");
      if (window.swLookup) window.swLookup(built.wrap);
      return built;
    }
    if (kind === "number") {
      built = control("number", el, id, name);
      built.input.step = "any";
      built.input.value = source || "";
      return built;
    }
    var multiline = MULTILINE[el.tagName] === 1 || el.textContent.indexOf("\n") >= 0 || (source !== null && source.indexOf("\n") >= 0);
    var max = el.getAttribute("data-max");
    if (multiline) {
      built = control(max ? "textarea-counted" : "textarea", el, id, name);
      var text = source !== null ? source : el.innerText.replace(/\n{3,}/g, "\n\n").trim();
      built.input.rows = Math.min(10, Math.max(3, text.split("\n").length + 1));
      built.input.setAttribute("tabindex", "0");
      built.input.value = text;
      return counted(built, id, max);
    }
    built = control(max ? "text-counted" : "text", el, id, name);
    built.input.value = source !== null ? source : el.textContent.trim();
    return counted(built, id, max);
  }

  // counted gives a field with a most it can hold its count, which the
  // text-field component's enhance.js keeps as a person types.
  function counted(built, id, max) {
    if (!max) return built;
    var count = built.wrap.querySelector(".sw-field__count");
    count.id = id + "-count";
    count.textContent = "Up to " + max + " characters";
    built.input.setAttribute("data-max", max);
    var by = (built.input.getAttribute("aria-describedby") || "").split(" ").filter(function (x) { return x && !/-count$/.test(x); });
    built.input.setAttribute("aria-describedby", by.concat(count.id).join(" "));
    if (window.swCount) window.swCount(built.wrap);
    return built;
  }

  // choose is one choice from a set, the select component either way: five
  // or fewer as radios, all in view, in a fieldset whose legend is the
  // field's name; more as a dropdown.
  function choose(el, id, name, current) {
    var options = [];
    try { options = JSON.parse(el.getAttribute("data-options")) || []; } catch (e) { options = []; }
    var built, i;
    if (options.length > 5) {
      built = control("dropdown", el, id, name);
      built.input.innerHTML = "";
      for (i = 0; i < options.length; i++) {
        var opt = document.createElement("option");
        opt.value = options[i].value;
        opt.textContent = options[i].label;
        opt.selected = options[i].value === current;
        built.input.appendChild(opt);
      }
      return built;
    }
    var held = document.getElementById("sw-controls").content.querySelector('[data-control="radios"]');
    var wrap = held.firstElementChild.cloneNode(true);
    wrap.classList.add("sw-inline-field");
    wrap.querySelector("legend").textContent = el.getAttribute("data-label") || label(name);
    var list = wrap.querySelector(".sw-choice__options");
    var row = list.firstElementChild;
    list.innerHTML = "";
    var focus = null;
    for (i = 0; i < options.length; i++) {
      var r = row.cloneNode(true);
      var input = r.querySelector("input");
      input.id = i ? id + "-" + i : id;
      input.name = "prop-" + name;
      input.value = options[i].value;
      input.checked = options[i].value === current;
      r.querySelector("label").setAttribute("for", input.id);
      r.querySelector("label").textContent = options[i].label;
      list.appendChild(r);
      if (!focus || input.checked) focus = input;
    }
    return { wrap: wrap, input: focus };
  }

  // edit replaces the marked elements of one block with a small form.
  function edit(block) {
    if (block.querySelector(".sw-inline-form")) return;
    var marked = (window.swEditFields && window.swEditFields(block)) || block.querySelectorAll("[data-prop]");
    if (!marked.length) return;
    var id = block.getAttribute("data-block-id");

    var form = document.createElement("form");
    form.className = "sw-inline-form sw-stack";
    form.method = "post";
    var action = block.getAttribute("data-edit-action");
    if (!action) { action = "/canvas/"+id+"/props"; }
    form.action = action;
    form.setAttribute("aria-label", "Edit " + (block.getAttribute("data-block-label") || "this block"));

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
    var dl = block.querySelector("dl.sw-fields");
    if (dl) {
      dl.hidden = true;
      dl.style.display = "none";
      covered.push(dl);
    }

    for (var c = form.nextElementSibling; c; c = c.nextElementSibling) {
      if (c.classList.contains("sw-visually-hidden")) continue;
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
    // 09-edit-fields.js tidies the form: empty fields behind Add buttons.
    document.dispatchEvent(new CustomEvent("sw:edit-open", { detail: { block: block, form: form } }));
  }

  function cancel(block, form, covered) {
    // A record made a moment ago and never saved goes back to the list instead (09-edit-fields.js).
    if (!document.dispatchEvent(new CustomEvent("sw:edit-cancel", { cancelable: true, detail: { block: block, form: form } }))) return;
    for (var i = 0; i < covered.length; i++) {
      covered[i].style.display = "";
      covered[i].hidden = false;
    }
    // Explicitly restore the definition list as well.
    var dl = block.querySelector("dl.sw-fields");
    if (dl) dl.hidden = false;
    form.remove();
    var btn = block.querySelector("[data-edit]");
    if (btn) btn.focus();
  }

  // Every block with editable text gets an Edit button, added here so it
  // never exists in a browser that cannot honour it.
  function arm(block) {
    if (!block.querySelector("[data-prop], template[data-edit-fields]")) return;
    var bar = block.querySelector(".sw-bar");
    if (!bar || bar.querySelector("[data-edit]")) return;
    var name = block.getAttribute("data-block-component") || "block";
    var label = block.getAttribute("data-block-label");
    if (!label) { label = name; }
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "sw-button sw-button--quiet sw-pressable";
    btn.setAttribute("data-edit", "");
    btn.innerHTML = 'Edit<span class="sw-visually-hidden"> ' + label + "</span>";
    btn.addEventListener("click", function () { edit(block); });
    bar.insertBefore(btn, bar.firstChild);
  }

  function init() {
    document.querySelectorAll("[data-block-id]").forEach(arm);
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  // A page that refreshed part of itself during a live turn has new blocks
  // to arm; the ones already armed say so and are left alone.
  document.addEventListener("sw:refresh", init);
})();
