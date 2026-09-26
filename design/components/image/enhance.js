// A picture that moves starts still, with a Play button beside it that
// makes it move and a Stop button that stills it again: nothing on the page
// moves on its own for more than a moment (WCAG 2.2.2). Without this script
// it stays still, and Open the original shows it moving.
//
// A picture that cannot be loaded says so in words beside it, where the
// browser shows only its broken-picture mark and the words it stands for.
(function () {
  "use strict";
  function arm(img) {
    if (img._armed) return;
    img._armed = true;
    img.addEventListener("error", function () {
      if (img.nextElementSibling && img.nextElementSibling.classList.contains("sw-image__missing")) return;
      var p = document.createElement("span");
      p.className = "sw-image__missing";
      p.textContent = "This picture could not be loaded.";
      img.insertAdjacentElement("afterend", p);
    });
    var moving = img.getAttribute("data-moving"), still = img.getAttribute("data-still");
    if (!moving || !still) return;
    var b = document.createElement("button");
    b.type = "button";
    b.className = "sw-button sw-button--secondary sw-pressable sw-image__play";
    b.setAttribute("aria-pressed", "false");
    b.textContent = "Play";
    b.addEventListener("click", function () {
      var play = img.getAttribute("src") === still;
      img.setAttribute("src", play ? moving : still);
      b.textContent = play ? "Stop" : "Play";
      b.setAttribute("aria-pressed", play ? "true" : "false");
    });
    img.insertAdjacentElement("afterend", b);
  }
  function init() { document.querySelectorAll("[data-component=image] .sw-image__img").forEach(arm); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
  document.addEventListener("sw:refresh", init);
})();
