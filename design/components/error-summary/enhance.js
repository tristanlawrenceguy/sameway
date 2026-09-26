// An error summary's links lead to their fields, and the fields say so.
//
// A problem's link names a field; pressing it moves focus to that field
// (a link to an input only scrolls there on its own), opening the record's
// editor first when it is closed. swErrorSummary marks each field a summary
// names as invalid and puts its problem beside it, in words, where it can be
// seen, not only in a red edge; the field is described by that. Then focus
// goes to the summary, once, so the whole list is heard first and each link
// leads on: the inline editor calls it when it opens an edit that was
// refused.
(function () {
  "use strict";
  function fieldFor(name) {
    return document.getElementById("field-" + name) ||
      document.querySelector('.sw-inline-form [name="prop-' + name + '"]');
  }
  // The field, opening the editor of the block that shows it when it is
  // not open yet.
  function openFor(name) {
    var field = fieldFor(name);
    if (field) return field;
    var shown = document.querySelector('[data-prop="' + name + '"]');
    var block = shown && shown.closest("[data-block-id]");
    var edit = block && block.querySelector("[data-edit]");
    if (edit) edit.click();
    return fieldFor(name);
  }
  // The problem beside its field, above the input, as every field's own
  // error is: "Error: Title is required."
  function besideField(field, text) {
    var id = (field.id || field.name) + "-error";
    if (document.getElementById(id)) return id;
    var p = document.createElement("p");
    p.className = "sw-field__error";
    p.id = id;
    var said = document.createElement("span");
    said.className = "sw-visually-hidden";
    said.textContent = "Error: ";
    p.appendChild(said);
    p.appendChild(document.createTextNode(text));
    var wrap = field.closest(".sw-field");
    var before = wrap ? wrap.querySelector("input, select, textarea, fieldset") : field;
    (before || field).parentNode.insertBefore(p, before || field);
    return id;
  }
  window.swErrorSummary = function () {
    document.querySelectorAll("[data-component=error-summary]").forEach(function (summary) {
      var marked = 0;
      summary.querySelectorAll("[data-field]").forEach(function (link) {
        var field = fieldFor(link.getAttribute("data-field"));
        if (!field) return;
        marked++;
        field.setAttribute("aria-invalid", "true");
        var said = besideField(field, link.textContent);
        var by = (field.getAttribute("aria-describedby") || "").split(" ").filter(function (x) { return x && x !== "outcome"; });
        if (by.indexOf(said) < 0) by.push(said);
        field.setAttribute("aria-describedby", by.join(" "));
      });
      if (marked && !summary._focused) {
        summary._focused = true;
        summary.focus();
      }
    });
  };
  document.addEventListener("click", function (e) {
    var link = e.target.closest && e.target.closest("[data-component=error-summary] [data-field]");
    if (!link) return;
    var field = openFor(link.getAttribute("data-field"));
    if (!field) return;
    e.preventDefault();
    window.swErrorSummary();
    field.focus();
  });
  function init() { window.swErrorSummary(); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
