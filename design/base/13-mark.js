// A mark saves itself, where it is. The checkbox is the whole control when
// scripts run: a change sends the form in the background, and the person
// stays on the box they ticked, to tick the next one, instead of the page
// reloading and focus going to its top (WCAG 3.2.2: changing a control
// does not change the context). The outcome, with its Undo, is shown where
// outcomes are, and said through a region that was there from the start,
// since one put in with its words is often not read out. A save the server
// refused puts the box back and says why. The list is brought up to date
// when focus leaves it, not under the person's hands: a ticked row would
// jump to the Done group, or out of a list of what is not done.
//
// Without scripts, or if the background send fails, the form is sent as a
// form, and its Save button, there for a browser without scripts, works.
(function () {
  "use strict";
  // The regions that say what happened, outside the part of the page a
  // refresh replaces, empty until there is something to say.
  function region(id, role) {
    var r = document.getElementById(id);
    if (r) return r;
    r = document.createElement("p");
    r.id = id;
    r.className = "sw-visually-hidden";
    r.setAttribute("role", role);
    document.body.appendChild(r);
    return r;
  }
  function say(fresh, failed) {
    var words = (fresh.querySelector(".sw-alert__message") || fresh).textContent.trim();
    var r = region(failed ? "sw-mark-failed" : "sw-mark-said", failed ? "alert" : "status");
    r.textContent = "";
    setTimeout(function () { r.textContent = words; }, 50);
    // The copy on the page is for the eye; the region has said it.
    fresh.querySelectorAll("[role=status], [role=alert]").forEach(function (x) { x.removeAttribute("role"); });
  }
  function show(fresh) {
    var old = document.getElementById("outcome");
    if (old) old.replaceWith(fresh);
    else {
      var head = document.querySelector("main .sw-page-head") || document.querySelector("main");
      if (head) head.parentNode === document.body ? head.prepend(fresh) : head.after(fresh);
    }
    document.dispatchEvent(new CustomEvent("sw:refresh"));
  }
  // Once focus leaves the list the box was in, the page catches up.
  function refreshWhenLeft(form) {
    var list = form.closest("ol, ul, .sw-collection, .sw-rows") || form.parentNode;
    if (list._waiting) return;
    list._waiting = true;
    list.addEventListener("focusout", function gone(e) {
      if (e.relatedTarget && list.contains(e.relatedTarget)) return;
      list.removeEventListener("focusout", gone);
      list._waiting = false;
      if (window.swRefresh) window.swRefresh(0);
    });
  }
  function arm(form) {
    if (form.classList.contains("sw-mark--live")) return;
    form.classList.add("sw-mark--live");
    var box = form.querySelector(".sw-mark__input");
    if (!box) return;
    var busy = false, again = false;
    function send() {
      busy = true;
      var sent = box.checked;
      // Sent as a form sends it, url-encoded: the server reads no multipart.
      fetch(form.action, { method: "POST", body: new URLSearchParams(new FormData(form)), credentials: "same-origin", headers: { "X-Requested-With": "sameway-mark" } })
        .then(function (r) { if (!r.ok) throw new Error(String(r.status)); return r.text(); })
        .then(function (html) {
          var t = document.createElement("template");
          t.innerHTML = html.trim();
          var fresh = t.content.firstElementChild;
          var failed = !!(fresh && fresh.matches("[data-outcome=failed]"));
          if (failed) box.checked = !sent;
          var row = form.closest(".sw-row");
          if (row) row.classList.toggle("sw-row--done", box.checked);
          if (fresh) { say(fresh, failed); show(fresh); }
          busy = false;
          // Ticked again while it saved: once more, as the box is now.
          if (!failed && (again || box.checked !== sent)) { again = false; if (box.checked !== sent) return send(); }
          if (!failed) refreshWhenLeft(form);
        })
        .catch(function () { busy = false; if (form.requestSubmit) form.requestSubmit(); else form.submit(); });
    }
    box.addEventListener("change", function () {
      if (!window.fetch) { form.submit(); return; }
      if (busy) { again = true; return; }
      send();
    });
  }
  function init() { document.querySelectorAll("form.sw-mark").forEach(arm); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
