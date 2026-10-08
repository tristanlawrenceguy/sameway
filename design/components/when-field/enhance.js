// A when-field's picker, and how the words were read.
//
// The picker is shown once there is a script to make it useful: picking a
// day writes it into the words, keeping any time typed. When the words are
// changed, the server, the one that reads them, says how: "Reads as Fri 2
// Oct 2026, 14:00", or what they must be; the picker follows. A repeat has
// no picker and is said back the same way: "Reads as every Tuesday". The
// words are what is sent. Without this script the picker stays hidden, the
// words work alone, and the saved message says how they were read.
(function () {
  "use strict";
  var MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  // A time in the words, kept when a day is picked: 2pm, 14:00, 14.30,
  // noon, midnight; not the dots of a date such as 19.09.2026.
  var TIME = /(^|\s)(\d{1,2}([:.]\d{2})?\s*(am|pm)|\d{1,2}[:.]\d{2}|noon|midday|midnight)(?=[\s,]|$)/i;

  // swWhenField arms every when-field in root; the inline editor calls it on
  // a form it has just built.
  window.swWhenField = function (root) {
    root = root || document;
    var fields = Array.prototype.slice.call(root.querySelectorAll("[data-component=when-field]"));
    if (root.matches && root.matches("[data-component=when-field]")) fields.unshift(root);
    fields.forEach(function (field) {
      var pick = field.querySelector(".sw-when-field__pick");
      var row = field.querySelector(".sw-when-field__pick-row");
      var words = field.querySelector("input[type=text]");
      var read = field.querySelector(".sw-when-field__read");
      var repeat = field.hasAttribute("data-repeat");
      if (!words || words._armed || (!pick && !repeat)) return;
      words._armed = true;
      if (row) row.hidden = false;
      // A press anywhere on the picker opens the month, not only its icon.
      if (pick) pick.addEventListener("click", function () {
        try { pick.showPicker(); } catch (e) { /* the browser opens it its own way */ }
      });
      if (pick) pick.addEventListener("change", function () {
        if (!pick.value) return;
        var p = pick.value.split("-");
        var day = Number(p[2]) + " " + MONTHS[Number(p[1]) - 1] + " " + p[0];
        var time = (words.value.match(TIME) || [])[2];
        words.value = time ? day + " " + time : day;
        tell();
      });
      // How the words were read, said once they are changed, not on every key.
      function tell() {
        if (!read) return;
        var q = words.value.trim();
        if (!q) { read.hidden = true; read.textContent = ""; return; }
        fetch("/when?" + (repeat ? "repeat=" : "words=") + encodeURIComponent(q), { headers: { accept: "application/json" } })
          .then(function (r) { return r.json(); })
          .then(function (out) {
            read.textContent = out.text ? "Reads as " + out.text : "Not read: " + (out.error || "");
            read.hidden = false;
            if (out.day && pick) pick.value = out.day;
            var by = (words.getAttribute("aria-describedby") || "").split(" ").filter(Boolean);
            if (by.indexOf(read.id) < 0) words.setAttribute("aria-describedby", by.concat(read.id).join(" "));
          })
          .catch(function () { /* the server says so when it is saved */ });
      }
      words.addEventListener("change", tell);
    });
  };
  sw.arm("[data-component=when-field]", window.swWhenField);
})();
