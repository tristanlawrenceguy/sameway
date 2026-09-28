// A lookup offers matching records as a person types, and sends the id of
// the one chosen.
//
// The box shows names; a hidden field carries the id. The suggestions are
// a list drawn here, not the browser's own datalist, which no desktop
// browser enlarges when the page is zoomed, some phones do not show, and
// voice control cannot open. It follows the APG list autocomplete: Down
// and Up move through it, Enter chooses, Escape closes it and then puts
// back what was chosen before. What was found is said as it changes, and
// a name that is not one of the records is said to be so, not saved as
// the record chosen before. Without this script the box shows the name
// and the id it came with is sent unchanged.
(function () {
  "use strict";
  var said = 0;
  function el(tag, cls) { var e = document.createElement(tag); if (cls) e.className = cls; return e; }

  function arm(field) {
    var box = field.querySelector("input[type=text]");
    var chosen = field.querySelector("input[type=hidden]");
    if (!box || !chosen || box._armed) return;
    box._armed = true;
    var to = field.getAttribute("data-to") || "";
    var what = field.getAttribute("data-what") || to || "record";
    var label = field.querySelector("label");
    var n = ++said;
    // The browser's own list goes; this one takes its place.
    box.removeAttribute("list");
    var old = field.querySelector("datalist");
    if (old) old.remove();
    var list = el("ul", "sw-plain sw-lookup__list");
    list.id = box.id + "-options";
    list.setAttribute("role", "listbox");
    list.hidden = true;
    if (label) { label.id = label.id || box.id + "-label"; list.setAttribute("aria-labelledby", label.id); }
    var status = el("p", "sw-visually-hidden");
    status.setAttribute("aria-live", "polite");
    var error = el("p", "sw-field__error");
    error.id = box.id + "-lookup-error";
    error.hidden = true;
    box.insertAdjacentElement("afterend", list);
    list.insertAdjacentElement("afterend", status);
    box.insertAdjacentElement("beforebegin", error);
    box.setAttribute("role", "combobox");
    box.setAttribute("aria-autocomplete", "list");
    box.setAttribute("aria-expanded", "false");
    box.setAttribute("aria-controls", list.id);

    var before = { id: chosen.value, title: box.value };
    var hits = [], active = -1, seq = 0, wait, sayLater;

    function say(words) {
      clearTimeout(sayLater);
      sayLater = setTimeout(function () { status.textContent = words; }, 500);
    }
    function open(yes) {
      list.hidden = !yes;
      box.setAttribute("aria-expanded", yes ? "true" : "false");
      if (!yes) { active = -1; box.removeAttribute("aria-activedescendant"); }
    }
    function highlight(i) {
      var opts = list.querySelectorAll("[role=option]");
      if (!opts.length) return;
      active = (i + opts.length) % opts.length;
      opts.forEach(function (o, k) { o.setAttribute("aria-selected", k === active ? "true" : "false"); });
      box.setAttribute("aria-activedescendant", opts[active].id);
      opts[active].scrollIntoView({ block: "nearest" });
    }
    function problem(words) {
      error.hidden = !words;
      error.textContent = words || "";
      var by = (box.getAttribute("aria-describedby") || "").split(" ").filter(function (x) { return x && x !== error.id; });
      if (words) { by.push(error.id); box.setAttribute("aria-invalid", "true"); } else box.removeAttribute("aria-invalid");
      box.setAttribute("aria-describedby", by.join(" "));
    }
    function choose(h) {
      chosen.value = h.id;
      box.value = h.title;
      before = { id: h.id, title: h.title };
      problem("");
      open(false);
      chosen.dispatchEvent(new Event("change", { bubbles: true }));
    }
    function show(found, q) {
      hits = found;
      list.innerHTML = "";
      // Two records with one title are told apart by a line of each.
      var titles = {};
      hits.forEach(function (h) { titles[h.title] = (titles[h.title] || 0) + 1; });
      hits.forEach(function (h, i) {
        var o = el("li", "sw-lookup__option");
        o.id = list.id + "-" + i;
        o.setAttribute("role", "option");
        o.setAttribute("aria-selected", "false");
        o.textContent = h.title;
        if (titles[h.title] > 1 && h.snippet) {
          var more = el("span", "sw-lookup__more");
          more.textContent = h.snippet;
          o.appendChild(more);
        }
        o.addEventListener("pointerdown", function (e) { e.preventDefault(); choose(h); });
        list.appendChild(o);
      });
      if (!hits.length) {
        var none = el("li", "sw-lookup__none");
        none.textContent = "No " + what + " matches “" + q + "”";
        list.appendChild(none);
      }
      open(true);
      say(hits.length ? hits.length + (hits.length === 1 ? " " + what + " found" : " found") : "No " + what + " matches " + q);
    }
    function look() {
      var q = box.value.trim();
      clearTimeout(wait);
      if (!q) { chosen.value = ""; problem(""); open(false); return; }
      if (q.length < 2) { open(false); say("Type 2 or more letters"); return; }
      var mine = ++seq;
      wait = setTimeout(function () {
        fetch("/api/search?type=" + encodeURIComponent(to) + "&q=" + encodeURIComponent(q), { headers: { accept: "application/json" } })
          .then(function (r) { if (!r.ok) throw new Error(r.status); return r.json(); })
          .then(function (found) { if (mine === seq) show((found.hits || []).slice(0, 20), q); })
          .catch(function () { if (mine === seq) say("Could not search just now. Try again."); });
      }, 200);
    }
    box.addEventListener("input", function () {
      // The name no longer the one chosen: nothing is chosen until one is.
      if (box.value !== before.title) chosen.value = "";
      look();
    });
    box.addEventListener("keydown", function (e) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        if (list.hidden) { if (hits.length) open(true); else look(); return; }
        highlight(active + (e.key === "ArrowDown" ? 1 : -1));
      } else if (e.key === "Enter" && !list.hidden && active >= 0) {
        e.preventDefault();
        choose(hits[active]);
      } else if (e.key === "Escape") {
        // The editor's Escape cancels the whole form; the first one here
        // only closes the list, the second puts back what was chosen.
        if (!list.hidden) { e.preventDefault(); e.stopPropagation(); open(false); return; }
        if (box.value !== before.title) { e.preventDefault(); e.stopPropagation(); box.value = before.title; chosen.value = before.id; problem(""); }
      }
    });
    box.addEventListener("blur", function () {
      open(false);
      var q = box.value.trim();
      if (!q) { chosen.value = ""; problem(""); return; }
      if (!chosen.value) {
        var exact = hits.filter(function (h) { return h.title === q; });
        if (exact.length === 1) { choose(exact[0]); return; }
        problem("No " + what + " called “" + q + "”. Choose one from the list, or clear the box.");
      }
    });
    field.setAttribute("data-lookup", String(n));
  }

  // swLookup arms every lookup in root; the inline editor calls it on a
  // field it has just made.
  window.swLookup = function (root) {
    root = root || document;
    var all = Array.prototype.slice.call(root.querySelectorAll("[data-component=lookup]"));
    if (root.matches && root.matches("[data-component=lookup]")) all.unshift(root);
    all.forEach(arm);
  };
  function init() { window.swLookup(document); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
