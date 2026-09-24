// A lookup offers matching records as a person types, from /api/search,
// and sends the id of the one chosen.
//
// The box shows names; a hidden field carries the id. A name typed that
// matches a suggestion exactly chooses it; clearing the box chooses none.
(function () {
  "use strict";
  // swLookup arms every lookup in root; the inline editor calls it on a
  // field it has just made.
  window.swLookup = function (root) {
    root = root || document;
    var all = Array.prototype.slice.call(root.querySelectorAll("[data-component=lookup]"));
    if (root.matches && root.matches("[data-component=lookup]")) all.unshift(root);
    all.forEach(function (field) {
      var box = field.querySelector("input[type=text]");
      var id = field.querySelector("input[type=hidden]");
      var list = field.querySelector("datalist");
      if (!box || !id || !list || box._armed) return;
      box._armed = true;
      var to = field.getAttribute("data-to");
      var ids = {};
      var wait;
      function choose() {
        if (!box.value.trim()) { id.value = ""; return; }
        if (ids[box.value] !== undefined) id.value = ids[box.value];
      }
      box.addEventListener("input", function () {
        choose();
        clearTimeout(wait);
        var q = box.value.trim();
        if (q.length < 2) return;
        wait = setTimeout(function () {
          fetch("/api/search?q=" + encodeURIComponent(q), { headers: { accept: "application/json" } })
            .then(function (r) { return r.json(); })
            .then(function (found) {
              list.innerHTML = "";
              ids = {};
              (found.hits || []).filter(function (h) { return h.type === to; }).slice(0, 20).forEach(function (h) {
                var opt = document.createElement("option");
                opt.value = h.title;
                ids[h.title] = h.id;
                list.appendChild(opt);
              });
              choose();
            })
            .catch(function () { /* no suggestions this time; typing still works */ });
        }, 200);
      });
      box.addEventListener("change", choose);
    });
  };
  function init() { window.swLookup(document); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
