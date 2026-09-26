// A mark saves itself, where it is. The checkbox is the whole control when
// scripts run: a change sends the form in the background, and the person
// stays on the box they ticked, to tick the next one, instead of the page
// reloading and focus going to its top (WCAG 3.2.2: changing a control
// does not change the context). The outcome, with its Undo, is shown where
// outcomes are and said politely, and the row shows itself done.
//
// Without scripts, or if the background send fails, the form is sent as a
// form, and its Save button, there for a browser without scripts, works.
(function () {
  "use strict";
  function show(html) {
    var t = document.createElement("template");
    t.innerHTML = html.trim();
    var fresh = t.content.firstElementChild;
    if (!fresh) return;
    var old = document.getElementById("outcome");
    if (old) old.replaceWith(fresh);
    else {
      var main = document.querySelector("main .sw-page-head") || document.querySelector("main");
      if (main) main.parentNode === document.body ? main.prepend(fresh) : main.after(fresh);
    }
    document.dispatchEvent(new CustomEvent("sw:refresh"));
  }
  function arm(form) {
    if (form.classList.contains("sw-mark--live")) return;
    form.classList.add("sw-mark--live");
    var box = form.querySelector(".sw-mark__input");
    if (!box) return;
    box.addEventListener("change", function () {
      if (!window.fetch) { form.submit(); return; }
      // Sent as a form sends it, url-encoded: the server reads no multipart.
      fetch(form.action, { method: "POST", body: new URLSearchParams(new FormData(form)), credentials: "same-origin", headers: { "X-Requested-With": "sameway-mark" } })
        .then(function (r) { if (!r.ok) throw new Error(String(r.status)); return r.text(); })
        .then(function (html) {
          var row = form.closest(".sw-row");
          if (row) row.classList.toggle("sw-row--done", box.checked);
          show(html);
          if (window.swRefresh) window.swRefresh(600);
        })
        .catch(function () { if (form.requestSubmit) form.requestSubmit(); else form.submit(); });
    });
  }
  function init() { document.querySelectorAll("form.sw-mark").forEach(arm); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
