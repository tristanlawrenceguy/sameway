// Log saves itself, where it is. With scripts, pressing Log sends the
// amount in the background: focus stays on Log, to press it again for the
// next glass, instead of the page reloading to its top; what was logged and
// where it stands now are said, with Undo, where outcomes are (13-mark.js
// does the same for a box ticked); the list catches up when focus leaves
// it. Without scripts, or if the send fails, the form is sent as a form.
(function () {
  "use strict";
  function region(id, role) {
    var r = document.getElementById(id);
    if (r) return r;
    r = document.createElement("p");
    r.id = id; r.className = "sw-visually-hidden"; r.setAttribute("role", role);
    document.body.appendChild(r);
    return r;
  }
  function show(fresh, failed) {
    // What was logged and where it stands: the title and the words under it.
    var said = fresh.querySelector(".sw-error-summary") || fresh.querySelector(".sw-alert");
    var parts = said ? said.querySelectorAll(".sw-alert__title, .sw-alert__message, .sw-error-summary__title, li") : [];
    // What a screen reader would read there: marks hidden from it are left out.
    var words = Array.prototype.map.call(parts, function (x) {
      var c = x.cloneNode(true);
      c.querySelectorAll("[aria-hidden=true]").forEach(function (h) { h.remove(); });
      return c.textContent.trim();
    }).join(" ") || fresh.textContent;
    words = words.replace(/\s+/g, " ").trim();
    var r = region(failed ? "sw-log-failed" : "sw-log-said", failed ? "alert" : "status");
    r.textContent = "";
    setTimeout(function () { r.textContent = words; }, 50);
    fresh.querySelectorAll("[role=status], [role=alert]").forEach(function (x) { x.removeAttribute("role"); });
    var old = document.getElementById("outcome");
    if (old) old.replaceWith(fresh);
    else {
      var head = document.querySelector("main .sw-page-head") || document.querySelector("main");
      if (head) head.parentNode === document.body ? head.prepend(fresh) : head.after(fresh);
    }
    sw.emit("refresh");
  }
  function arm(form) {
    if (form._logArmed || !window.fetch) return;
    form._logArmed = true;
    form.addEventListener("submit", function (e) {
      e.preventDefault();
      var list = form.closest(".sw-tracker") || form.parentNode;
      fetch(form.action, { method: "POST", body: new URLSearchParams(new FormData(form)), credentials: "same-origin", headers: { "X-Requested-With": "sameway-mark" } })
        .then(function (r) { if (!r.ok) throw new Error(String(r.status)); return r.text(); })
        .then(function (html) {
          var t = document.createElement("template");
          t.innerHTML = html.trim();
          var fresh = t.content.firstElementChild;
          if (!fresh) return;
          var failed = fresh.matches("[data-outcome=failed]");
          show(fresh, failed);
          if (failed) return;
          var amount = form.querySelector(".sw-tracker__amount");
          if (amount && !amount.defaultValue) amount.value = "";
          if (list._waiting) return;
          list._waiting = true;
          list.addEventListener("focusout", function gone(ev) {
            if (ev.relatedTarget && list.contains(ev.relatedTarget)) return;
            list.removeEventListener("focusout", gone);
            list._waiting = false;
            sw.refresh(0);
          });
        })
        .catch(function () { form.submit(); });
    });
  }
  function init() { document.querySelectorAll("form.sw-tracker__log").forEach(arm); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
  document.addEventListener("sw:refresh", init);
})();
