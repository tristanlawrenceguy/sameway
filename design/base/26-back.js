// Back where the action was taken.
//
// A form on the canvas already says which block it is in (back.go), so
// without this script the page comes back to that block. This narrows it
// to the nearest thing with an id around the form, such as the row or
// card, and on arrival moves focus there, so a keyboard or a screen reader
// carries on from the same place rather than the top of the page. An
// error summary, when there is one, keeps focus: what went wrong comes
// first.
//
// A form may name its own control to come back to, as a board's Move
// does: that is closer still, so it is kept. Focus then returns to the
// very thing pressed, and what the action came to is read with it, once,
// rather than focus going to the message at the top of the page, far from
// where the person was working (the next card to move).
(function () {
  "use strict";
  document.addEventListener("submit", function (e) {
    var form = e.target;
    if (!form || (form.method || "").toLowerCase() !== "post") return;
    var near = form.closest("[id]");
    if (!near || near.id === "main" || near.classList.contains("sw-block")) return;
    var named = form.querySelectorAll("input[name=back]");
    for (var i = 0; i < named.length; i++) {
      var at = document.getElementById(named[i].value);
      if (at && at !== near && near.contains(at)) return;
    }
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
    var control = place.matches("a[href], button, input, select, textarea");
    if (!control && !place.matches("[tabindex]")) place.setAttribute("tabindex", "-1");
    place.focus({ preventScroll: true });
    if (control) tell(place);
  }

  // A control come back to hears what its action came to as its
  // description, once: the outcome keeps its place, and its Undo, at the
  // top (alert/enhance.js leaves focus here when told).
  function tell(control) {
    var outcome = document.getElementById("outcome");
    var words = outcome && outcome.getAttribute("data-outcome") === "done" && outcome.querySelector(".sw-alert__message");
    if (!words || document.activeElement !== control) return;
    words.id = words.id || "outcome-words";
    var had = control.getAttribute("aria-describedby");
    control.setAttribute("aria-describedby", words.id);
    outcome.setAttribute("data-outcome-told", "");
    control.addEventListener("blur", function () {
      if (had) control.setAttribute("aria-describedby", had); else control.removeAttribute("aria-describedby");
    }, { once: true });
  }
  sw.ready(arrive);
})();
