// A proposal's answer is sent without leaving the page.
(function () {
  "use strict";

  // A proposal's two buttons are forms that work on their own; with
  // scripts the answer is sent without leaving the page, and every
  // pending proposal goes with it, so at most one is ever waiting.
  function proposeHandler() {
    var btns = document.querySelectorAll(".sw-proposal [type=\"submit\"]");
    if (!btns || btns.length === 0) return;
    for (var i = 0; i < btns.length; i++) {
      (function (btn) {
        if (btn._proposeArmed) return;
        btn._proposeArmed = true;
        btn.addEventListener("click", function (e) {
          e.preventDefault();
          var form = btn.closest("form");
          fetch(form.action, { method: "POST" }).then(function () {
            document.querySelectorAll(".sw-proposal").forEach(function (p) { p.remove(); });
          });
        });
      })(btns[i]);
    }
  }

  function init() { proposeHandler(); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  // Parts refreshed during a live turn are armed too; armed ones say so.
  document.addEventListener("sw:refresh", init);
})();
