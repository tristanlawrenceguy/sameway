// A mark saves itself. The checkbox is the whole control when scripts run:
// a change submits the form, and the Save button beside it, which is there
// for a browser without scripts, is not needed. Nothing here exists for a
// browser that cannot honour it: the form works as a form.
(function () {
  "use strict";
  function arm(form) {
    if (form.classList.contains("sw-mark--live")) return;
    form.classList.add("sw-mark--live");
    var box = form.querySelector(".sw-mark__input");
    if (!box) return;
    box.addEventListener("change", function () {
      if (form.requestSubmit) form.requestSubmit();
      else form.submit();
    });
  }
  function init() { document.querySelectorAll("form.sw-mark").forEach(arm); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
