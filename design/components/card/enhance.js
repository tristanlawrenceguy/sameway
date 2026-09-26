// A card with a link: a press anywhere on it follows the link, as people
// expect of a card, while the title stays the one link a screen reader or
// a keyboard meets, and the words stay selectable.
//
// Not a press: dragging to select words, a long press, or a press on a
// control inside the card, which does its own thing. Ctrl or Cmd opens the
// page in a new tab, as it would on the link itself. Without this script
// the title alone is the link, a whole target.
(function () {
  "use strict";
  function arm(root) {
    (root || document).querySelectorAll("[data-component=card]").forEach(function (card) {
      var link = card.querySelector(".sw-card__title a");
      if (!link || card._armed) return;
      card._armed = true;
      card.setAttribute("data-press", "");
      var down = 0;
      card.addEventListener("pointerdown", function () { down = Date.now(); });
      card.addEventListener("click", function (e) {
        if (e.target.closest("a, button, input, select, textarea, label, summary")) return;
        if (Date.now() - down > 400 || String(window.getSelection() || "")) return;
        if (e.ctrlKey || e.metaKey) window.open(link.href, "_blank");
        else link.click();
      });
    });
  }
  function init() { arm(document); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
  document.addEventListener("sw:refresh", init);
})();
