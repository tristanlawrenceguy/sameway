// An error summary's links lead to their fields, and the fields say so.
//
// A problem's link names a field; pressing it moves focus to that field
// (a link to an input only scrolls there on its own). swErrorSummary marks
// each field a summary names as invalid, described by its problem, once the
// form holding it is on the page: the inline editor calls it when it opens
// an edit that was refused.
(function () {
  "use strict";
  function fieldFor(name) {
    return document.getElementById("field-" + name) ||
      document.querySelector('.sw-inline-form [name="prop-' + name + '"], [name="' + name + '"]');
  }
  window.swErrorSummary = function () {
    document.querySelectorAll("[data-component=error-summary] [data-field]").forEach(function (link) {
      var field = fieldFor(link.getAttribute("data-field"));
      if (!field) return;
      field.setAttribute("aria-invalid", "true");
      var said = link.parentNode.id;
      var by = (field.getAttribute("aria-describedby") || "").split(" ").filter(Boolean);
      if (said && by.indexOf(said) < 0) field.setAttribute("aria-describedby", by.concat(said).join(" "));
    });
  };
  document.addEventListener("click", function (e) {
    var link = e.target.closest && e.target.closest("[data-component=error-summary] [data-field]");
    if (!link) return;
    var field = fieldFor(link.getAttribute("data-field"));
    if (!field) return;
    e.preventDefault();
    field.focus();
  });
  function init() { window.swErrorSummary(); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
