// Arrival, for a person who has already caught up.
//
// Changes arrive on the page one after another, with time to take each in
// (see 03-motion.css). Someone who has understood should not have to wait:
// any key or click on the page ends the sequence and shows everything at
// once, and while it runs a Show all button says so. The sequence is CSS,
// so without this script it simply plays out; the script only shortens it.
// This is an enhancement and adds its own control, which is gone the moment
// there is nothing left to show.
(function () {
  "use strict";
  if (!document.getAnimations) return;

  function arriving() {
    return document.getAnimations().filter(function (a) {
      var name = a.animationName || "";
      return name.indexOf("sw-arrive") === 0 || name.indexOf("sw-edit") === 0 || name === "sw-glow";
    });
  }

  function showAll() {
    arriving().forEach(function (a) { a.finish(); });
    var btn = document.querySelector("[data-show-all]");
    if (btn) btn.remove();
  }

  function init() {
    var canvas = document.querySelector(".sw-main .sw-canvas:not(.sw-canvas--strip)") || document.querySelector(".sw-canvas");
    if (!canvas || !document.querySelector("[data-arrival]")) return;
    var running = arriving();
    if (running.length === 0) return;

    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "sw-button sw-button--quiet sw-pressable";
    btn.setAttribute("data-show-all", "");
    btn.textContent = "Show all";
    btn.addEventListener("click", showAll);
    canvas.parentNode.insertBefore(btn, canvas);

    // Any interaction with the page means the person is ready.
    var end = function () { showAll(); };
    document.addEventListener("keydown", end, { once: true });
    document.addEventListener("pointerdown", end, { once: true });

    // When the last animation ends on its own, the button goes with it.
    Promise.all(running.map(function (a) { return a.finished; })).then(function () {
      if (btn.parentNode) btn.remove();
    }, function () {});
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
