// Back where the action was taken.
//
// A form on the canvas already says which block it is in (back.go), so
// without this script the page comes back to that block. This narrows it
// to the nearest thing with an id around the form, such as the row or
// card, and on arrival moves focus there, so a keyboard or a screen reader
// carries on from the same place rather than the top of the page. An
// error summary, when there is one, keeps focus: what went wrong comes
// first.
(function () {
  "use strict";
  document.addEventListener("submit", function (e) {
    var form = e.target;
    if (!form || (form.method || "").toLowerCase() !== "post") return;
    var near = form.closest("[id]");
    if (!near || near.id === "main" || near.classList.contains("sw-block")) return;
    var input = document.createElement("input");
    input.type = "hidden";
    input.name = "back";
    input.value = near.id;
    form.appendChild(input);
  }, true);

  function arrive() {
    var id = decodeURIComponent((location.hash || "").slice(1));
    if (!id || id === "main" || document.querySelector("[data-component=error-summary]")) return;
    var place = document.getElementById(id);
    if (!place || place.contains(document.activeElement) && document.activeElement !== document.body) return;
    if (!place.matches("a[href], button, input, select, textarea, [tabindex]")) place.setAttribute("tabindex", "-1");
    place.focus({ preventScroll: true });
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", arrive);
  else arrive();
})();
