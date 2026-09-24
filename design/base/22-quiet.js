// Controls a pointer reveals can be put away (WCAG 1.4.13).
//
// A block's buttons fade in over its corner when the pointer is on it
// (04-quiet.css). Escape puts them away again, for someone reading with a
// magnifier the bar has covered; they come back when the pointer leaves
// and returns. Escape inside an edit is the edit's, and left alone.
(function () {
  "use strict";
  document.addEventListener("keydown", function (e) {
    if (e.key !== "Escape" || document.querySelector(".sw-inline-form")) return;
    var over = document.querySelectorAll(".sw-reveal:hover");
    if (!over.length) return;
    var group = over[over.length - 1];
    group.setAttribute("data-quiet-away", "");
    group.addEventListener("mouseleave", function () { group.removeAttribute("data-quiet-away"); }, { once: true });
  });
})();
