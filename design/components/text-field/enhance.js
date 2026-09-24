// A field with a most it can hold counts down as a person types.
//
// The field says "Up to 80 characters" without scripts. With them, the
// words under it say how many are left, or how many too many, as they
// type; a screen reader hears it once they pause, not on every key. The
// field does not stop them mid-word: the server says so if it is too long.
// Serves the textarea component too, which carries the same count.
(function () {
  "use strict";
  function said(n) { return n === 1 ? "1 character" : n + " characters"; }
  window.swCount = function (root) {
    root = root || document;
    root.querySelectorAll("[data-max]").forEach(function (field) {
      if (field._counted) return;
      // Looked for in root first: the editor counts a field before it is placed.
      var count = root.querySelector("#" + CSS.escape(field.id + "-count")) || document.getElementById(field.id + "-count");
      if (!count) return;
      field._counted = true;
      var max = Number(field.getAttribute("data-max"));
      // What a screen reader hears, a moment after typing stops.
      var live = document.createElement("span");
      live.className = "sw-visually-hidden";
      live.setAttribute("aria-live", "polite");
      count.parentNode.insertBefore(live, count.nextSibling);
      var wait;
      function update() {
        var left = max - field.value.length;
        var text = left >= 0 ? said(left) + " left" : said(-left) + " too many";
        count.textContent = text;
        count.classList.toggle("sw-field__count--over", left < 0);
        clearTimeout(wait);
        wait = setTimeout(function () { live.textContent = text; }, 1000);
      }
      field.addEventListener("input", update);
      count.textContent = said(max - field.value.length) + " left";
    });
  };
  function init() { window.swCount(document); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
