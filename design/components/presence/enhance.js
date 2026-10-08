// Progressive enhancement for the presence component.
//
// The line is read where it sits and is not a live region: people come
// and go, and hearing each would interrupt. One arrival is said, politely,
// because it changes what the person does next: someone coming onto the
// page they are on ("Hana is on this page too"). Each person is said at
// most once while the page is open, however often they come and go, and
// leaving is never said. What was here when the page loaded is not said
// either; it is in the line.
//
// The words go into a status that is on the page from the start, empty,
// outside the header the page replaces when it follows a change: a region
// put on the page with its words already in is not reliably read out.
(function () {
  "use strict";
  var heard = {}, region = null;
  function namesHere() {
    var out = [];
    document.querySelectorAll('[data-component="presence"] [data-here] .sw-person').forEach(function (p) {
      var name = p.textContent.replace(/\s+/g, " ").trim();
      if (name) out.push(name);
    });
    return out;
  }
  function start() {
    region = document.createElement("div");
    region.className = "sw-visually-hidden";
    region.setAttribute("role", "status");
    region.setAttribute("data-presence-said", "");
    document.body.appendChild(region);
    namesHere().forEach(function (n) { heard[n] = true; });
  }
  sw.on("refresh", function () {
    if (!region) return;
    var fresh = namesHere().filter(function (n) { return !heard[n]; });
    if (!fresh.length) return;
    fresh.forEach(function (n) { heard[n] = true; });
    var who = fresh.length === 1 ? fresh[0] : fresh.slice(0, -1).join(", ") + " and " + fresh[fresh.length - 1];
    region.textContent = who + (fresh.length === 1 ? " is" : " are") + " on this page too.";
    clearTimeout(region._clear);
    region._clear = setTimeout(function () { region.textContent = ""; }, 10000);
  });
  sw.ready(start);
})();
