// Progressive enhancement for the status component.
//
// How a status changes, its words, look and what is read out, is sw.status
// in the page core (design/base/00-sw.js), which every script uses.
//
// Any form with data-busy-target="<status id>" gets, on submit:
//   - aria-busy="true" on the form and data-state="working" on the nearest
//     [data-region], so agents driving the page can see a request is in flight
//   - the named status switched to "working" with data-busy-message
//   - its submit buttons marked aria-disabled to prevent a double send
//   - the document title prefixed so the tab shows progress
// Without JavaScript the form still submits; the server renders the outcome.
(function () {
  "use strict";
  function upgrade(form) {
    if (form._busyArmed || !document.getElementById(form.getAttribute("data-busy-target"))) return;
    form._busyArmed = true;
    form.addEventListener("submit", function (event) {
      if (form.getAttribute("aria-busy") === "true") { event.preventDefault(); return; }
      // Looked up now, not when the page loaded: a turn shown as it
      // happens puts a fresh status in place of the old one.
      var status = document.getElementById(form.getAttribute("data-busy-target"));
      if (!status) return;
      form.setAttribute("aria-busy", "true");
      var region = form.closest("[data-region]");
      if (region) region.setAttribute("data-state", "working");
      sw.status(status, "working", form.getAttribute("data-busy-message") || "Working…", "");
      form.querySelectorAll("button[type=submit]").forEach(function (b) { b.setAttribute("aria-disabled", "true"); });
      document.title = "⏳ " + document.title;
    });
  }
  sw.arm("form[data-busy-target]", upgrade);
  // A page brought back from the back/forward cache mid-request is whatever
  // it was when left: the server knows how it ended, so ask it again.
  window.addEventListener("pageshow", function (e) {
    if (e.persisted && document.querySelector("form[aria-busy=true]")) location.reload();
  });
})();
