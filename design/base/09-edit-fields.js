// The whole record, edited where it is.
//
// A record's page says the least it can at rest, so what it shows is not
// what can be changed: the title is the heading, a few fields are chips,
// an empty field is not there at all. The page carries what can be
// edited in a template it does not show (editfields.go): every field of
// the type in its order, marked the way 08-edit.js reads a field. Edit
// builds the form from that, so a person can rename a record, set its
// date, choose its project or fill in a field it never had, by hand, in
// place, without asking.
(function () {
  "use strict";

  // swEditFields is what the block says can be edited, or null when it
  // says nothing and the marked elements on it are what there is.
  window.swEditFields = function (block) {
    var t = block.querySelector("template[data-edit-fields]");
    if (!t || !t.content || !t.content.children.length) return null;
    return Array.prototype.slice.call(t.content.children);
  };

  // A record just made by hand arrives at #edit, open in its editor, its
  // name ready to change; the address is put back, so a reload does not
  // open it again.
  function openNew() {
    if (location.hash !== "#edit") return;
    if (window.history && history.replaceState) history.replaceState(null, "", location.pathname + location.search);
    setTimeout(function () {
      var edit = document.querySelector("[data-block-id] [data-edit]");
      if (edit) edit.click();
      var name = document.querySelector(".sw-inline-form input[type=text]");
      if (name && name.select) name.select();
    }, 0);
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", openNew); else openNew();

  // Cancel on a record added a moment ago and never saved takes the adding
  // back: the person changed their mind, and the list should not keep a
  // "New person" they did not want (add.go).
  document.addEventListener("sw:edit-cancel", function (e) {
    var url = e.detail.block.getAttribute("data-discard");
    if (!url) return;
    e.preventDefault();
    var f = document.createElement("form");
    f.method = "post";
    f.action = url;
    document.body.appendChild(f);
    f.submit();
  });

  // empty says whether a field holds nothing yet. A yes or no always holds
  // one of the two, so it is never empty.
  function empty(wrap) {
    var editor = wrap.querySelector("[contenteditable=true]");
    if (editor) return !editor.textContent.trim();
    var radios = wrap.querySelectorAll("input[type=radio]");
    if (radios.length) return !Array.prototype.some.call(radios, function (r) { return r.checked && r.value; });
    if (wrap.querySelector("input[type=checkbox]")) return false;
    var input = wrap.querySelector("input:not([type=hidden]):not([type=date]), textarea, select");
    return !!input && !input.required && !input.value.trim();
  }

  function nameOf(wrap) {
    var l = wrap.querySelector("legend, .sw-field__label, label");
    return l ? l.textContent.replace(/\(required\)/, "").trim() : "field";
  }

  // An editor opens with what the record has; a field it has nothing in
  // waits behind a button with its name, one press from being filled.
  // A record just added shows every field, since filling them is the point.
  document.addEventListener("sw:edit-open", function (e) {
    var block = e.detail.block, form = e.detail.form;
    if (block.hasAttribute("data-discard")) return;
    var waiting = Array.prototype.filter.call(form.querySelectorAll(".sw-inline-field"), empty);
    if (!waiting.length) return;
    var row = document.createElement("div");
    row.className = "sw-cluster sw-edit-more";
    row.setAttribute("role", "group");
    row.setAttribute("aria-label", "Add a field");
    row.innerHTML = '<span class="sw-muted sw-small" aria-hidden="true">Add</span>';
    waiting.forEach(function (wrap) {
      wrap.hidden = true;
      var b = document.createElement("button");
      b.type = "button";
      b.className = "sw-button sw-button--quiet sw-pressable";
      b.textContent = nameOf(wrap);
      b.addEventListener("click", function () {
        wrap.hidden = false;
        b.remove();
        if (!row.querySelector("button")) row.remove();
        var input = wrap.querySelector("[contenteditable=true], input:not([type=hidden]), textarea, select");
        if (input) input.focus();
      });
      row.appendChild(b);
    });
    var actions = form.querySelector(":scope > .sw-cluster:last-child");
    form.insertBefore(row, actions);
  });
})();
