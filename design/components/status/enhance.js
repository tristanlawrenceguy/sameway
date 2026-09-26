// Progressive enhancement for the status component.
//
// swStatus(el, state, words, said) is the one way a status changes: its
// state, the look that goes with it, its words, and what is read out after
// them but not drawn, all together, in the region already on the page, so
// it is heard and never shows one state's mark beside another's words. It
// stays polite: a person who started the wait is listening for its end, and
// an interruption is not needed. A wait still going after fifteen seconds
// is said once, so a screen reader user knows it has not stopped.
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
  window.swStatus = function (el, state, words, said) {
    if (!el) return;
    if (state) {
      el.setAttribute("data-state", state);
      el.className = el.className.replace(/sw-status--\w+/, "sw-status--" + state);
    }
    el.setAttribute("aria-live", "polite");
    var text = el.querySelector(".sw-status__text");
    if (text && words != null) text.textContent = words;
    var hidden = el.querySelector(".sw-status__said");
    if (said && !hidden) {
      hidden = document.createElement("span");
      hidden.className = "sw-status__said sw-visually-hidden";
      el.appendChild(hidden);
    }
    if (hidden) hidden.textContent = said ? " " + said : "";
    clearTimeout(el._still);
    if ((state || el.getAttribute("data-state")) === "working" && state) {
      el._still = setTimeout(function () {
        if (el.getAttribute("data-state") === "working") window.swStatus(el, null, el.getAttribute("data-still") || "Still working…");
      }, 15000);
    }
  };
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
      window.swStatus(status, "working", form.getAttribute("data-busy-message") || "Working…", "");
      form.querySelectorAll("button[type=submit]").forEach(function (b) { b.setAttribute("aria-disabled", "true"); });
      document.title = "⏳ " + document.title;
    });
  }
  function init() { document.querySelectorAll("form[data-busy-target]").forEach(upgrade); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
  document.addEventListener("sw:refresh", init);
  // A page brought back from the back/forward cache mid-request is whatever
  // it was when left: the server knows how it ended, so ask it again.
  window.addEventListener("pageshow", function (e) {
    if (e.persisted && document.querySelector("form[aria-busy=true]")) location.reload();
  });
})();
