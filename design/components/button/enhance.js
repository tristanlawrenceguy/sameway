// A form is sent once. A second press of its button while the first is
// still on its way, from a slow connection, a habit of double clicking or
// a tremor, would create or delete twice, or run an action twice. So the
// button pressed is marked busy (aria-disabled, not disabled, so it keeps
// focus and its name and value still go with the form) and further presses
// of that form are ignored until the next page arrives. Held until then,
// not for a second: a short guard lets a slow request be sent again.
//
// Left alone: a form a script already sends itself (it prevented the
// submit), a search (a GET, harmless to repeat), and a form the status
// component guards (data-busy-target). The server should still be safe
// against a repeat; this spares the person the second one.
(function () {
  "use strict";
  document.addEventListener("submit", function (e) {
    var form = e.target;
    if (e.defaultPrevented || !form || form.hasAttribute("data-busy-target") || (form.method || "").toLowerCase() === "get") return;
    if (form._sending) { e.preventDefault(); return; }
    form._sending = true;
    var button = e.submitter;
    if (button) button.setAttribute("aria-disabled", "true");
    // A reply that is a download never leaves the page: after a while the
    // form can be sent again, so nobody is stuck.
    setTimeout(function () {
      form._sending = false;
      if (button) button.removeAttribute("aria-disabled");
    }, 15000);
  });
  // Coming back to the page with Back shows it as it was left: sendable.
  window.addEventListener("pageshow", function (e) {
    if (!e.persisted) return;
    document.querySelectorAll("form").forEach(function (f) { f._sending = false; });
    document.querySelectorAll("button[aria-disabled=true]").forEach(function (b) { b.removeAttribute("aria-disabled"); });
  });
})();
