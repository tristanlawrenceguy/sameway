// Progressive enhancement for the status component.
//
// Any form with data-busy-target="<status id>" gets, on submit:
//   - aria-busy="true" on the form and data-state="working" on the nearest
//     [data-region], so agents driving the page can see a request is in flight
//   - the named status switched to state "working" with data-busy-message
//   - its submit buttons marked aria-disabled to prevent a double send
//   - the document title prefixed so the tab shows progress
// Without JavaScript the form still submits; the server renders the outcome.
(function () {
  "use strict";
  function upgrade(form) {
    var status = document.getElementById(form.getAttribute("data-busy-target"));
    if (!status) return;
    form.addEventListener("submit", function (event) {
      if (form.getAttribute("aria-busy") === "true") { event.preventDefault(); return; }
      form.setAttribute("aria-busy", "true");
      var region = form.closest("[data-region]");
      if (region) region.setAttribute("data-state", "working");
      var message = form.getAttribute("data-busy-message") || "Working…";
      status.setAttribute("data-state", "working");
      status.className = status.className.replace(/sw-status--\w+/, "sw-status--working");
      var text = status.querySelector(".sw-status__text");
      if (text) text.textContent = message;
      form.querySelectorAll("button[type=submit]").forEach(function (b) { b.setAttribute("aria-disabled", "true"); });
      document.title = "⏳ " + document.title;
    });
  }
  function init() { document.querySelectorAll("form[data-busy-target]").forEach(upgrade); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
  // A page restored from the back/forward cache must not stay busy.
  window.addEventListener("pageshow", function (e) {
    if (!e.persisted) return;
    document.querySelectorAll("form[aria-busy=true]").forEach(function (f) { f.removeAttribute("aria-busy"); });
  });
})();
