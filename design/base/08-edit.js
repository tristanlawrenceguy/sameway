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
      var text = source !== null ? source : el.innerText.replace(/\n{3,}/g, "\n\n").trim();
      input.rows = Math.min(10, Math.max(3, text.split("\n").length + 1));
      input.value = text;
    } else {
      input.type = "text";
      input.value = el.hasAttribute("data-source") ? el.getAttribute("data-source") : el.textContent.trim();
    }
    wrap.appendChild(lab);
    wrap.appendChild(input);
    return { wrap: wrap, input: input };
  }

  var MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

  // A day or a moment is written the way a person says it and read by the
  // server: 19 Sep, next Friday, tomorrow 2pm. People know the day they
  // mean and type it faster than they find it; the platform's own picker
  // stands beside the words for anyone who would rather look at a month,
  // and picking a day writes it into the words, keeping any time typed.
  function dateField(el, id, name) {
    var wrap = document.createElement("div");
    wrap.className = "sw-field sw-inline-field sw-when";
    var lab = document.createElement("label");
    lab.className = "sw-field__label";
    lab.setAttribute("for", id);
    lab.textContent = label(name);
    var hint = document.createElement("p");
    hint.className = "sw-field__hint";
    hint.id = id + "-hint";
    hint.textContent = "A day, like 19 Sep or next Friday, with a time if there is one, like 2pm.";
    var input = document.createElement("input");
    input.className = "sw-field__input";
    input.id = id;
    input.name = "prop-" + name;
    input.type = "text";
    input.value = el.textContent.trim();
    input.setAttribute("aria-describedby", hint.id);
    var pick = document.createElement("input");
    pick.className = "sw-field__input sw-datepicker sw-when__pick";
    pick.type = "date";
    pick.setAttribute("aria-label", "Pick the day for " + label(name).toLowerCase());
    var source = el.getAttribute("data-source") || "";
    if (/^\d{4}-\d{2}-\d{2}/.test(source)) pick.value = source.slice(0, 10);
    pick.addEventListener("change", function () {
      if (!pick.value) return;
      var p = pick.value.split("-");
      var day = Number(p[2]) + " " + MONTHS[Number(p[1]) - 1] + " " + p[0];
      var time = (input.value.match(/\d{1,2}(:\d{2})?\s*(am|pm)\b|\d{1,2}:\d{2}/i) || [""])[0];
      input.value = time ? day + " " + time : day;
    });
    var row = document.createElement("div");
    row.className = "sw-when__row";
    row.appendChild(input);
    row.appendChild(pick);
    wrap.appendChild(lab);
    wrap.appendChild(hint);
    wrap.appendChild(row);
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

  // Enter in the chat composer sends, the way the Send button does: through
  // the form's own submit, so the required check, the busy state and the
  // double-send guard all see it. Shift+Enter starts a new line.
  function composeKeyHandler() {
    document.querySelectorAll("form.sw-compose textarea").forEach(function (textarea) {
      if (textarea._composeHandled) return;
      textarea._composeHandled = true;
      var form = textarea.closest("form.sw-compose");
      textarea.addEventListener("keydown", function (e) {
        if (e.key === "Enter" && !e.shiftKey && !e.isComposing) {
          e.preventDefault();
          if (form.requestSubmit) form.requestSubmit();
          else form.submit();
        }
      });
    });
  }

  // A proposal's two buttons are forms that work on their own; with
  // scripts the answer is sent without leaving the page, and every
  // pending proposal goes with it, so at most one is ever waiting.
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
