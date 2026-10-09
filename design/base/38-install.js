// Sameway as an app of its own.
//
// A browser that can install a site as an app (Chrome, Edge, and others
// on a computer or an Android phone) says so with beforeinstallprompt.
// The page keeps it and shows the Install button it has hidden for this
// (help.go); pressed, the browser asks the person itself. Installed, the
// button goes. A browser that cannot never shows it, and Help says the
// other ways, an iPhone's Share and Add to Home Screen among them.
(function () {
  "use strict";
  var asked = null;
  function buttons() { return document.querySelectorAll("[data-install]"); }
  window.addEventListener("beforeinstallprompt", function (e) {
    e.preventDefault();
    asked = e;
    buttons().forEach(function (b) { b.hidden = false; });
  });
  window.addEventListener("appinstalled", function () {
    asked = null;
    buttons().forEach(function (b) { b.hidden = true; });
  });
  // Each button its own listener, a new one armed when the page refreshes.
  sw.arm("[data-install]", function (b) {
    b.hidden = !asked;
    b.addEventListener("click", function () {
      if (!asked) return;
      asked.prompt();
      asked.userChoice.then(function () { asked = null; buttons().forEach(function (x) { x.hidden = true; }); });
    });
  });
})();
