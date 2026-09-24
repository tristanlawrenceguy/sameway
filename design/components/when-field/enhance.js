// A when-field's picker: shown once there is a script to make it useful,
// and picking a day writes it into the words, keeping any time typed.
//
// The words are what is sent; the picker sends nothing itself. Without this
// script the picker stays hidden and the words work alone.
(function () {
  "use strict";
  var MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

  // swWhenField arms every when-field in root; the inline editor calls it on
  // a form it has just built.
  window.swWhenField = function (root) {
    root = root || document;
    var fields = Array.prototype.slice.call(root.querySelectorAll("[data-component=when-field]"));
    if (root.matches && root.matches("[data-component=when-field]")) fields.unshift(root);
    fields.forEach(function (field) {
      var pick = field.querySelector(".sw-when-field__pick");
      var words = field.querySelector("input[type=text]");
      if (!pick || !words || pick._armed) return;
      pick._armed = true;
      pick.hidden = false;
      pick.addEventListener("change", function () {
        if (!pick.value) return;
        var p = pick.value.split("-");
        var day = Number(p[2]) + " " + MONTHS[Number(p[1]) - 1] + " " + p[0];
        var time = (words.value.match(/\d{1,2}(:\d{2})?\s*(am|pm)\b|\d{1,2}:\d{2}/i) || [""])[0];
        words.value = time ? day + " " + time : day;
      });
    });
  };
  function init() { window.swWhenField(document); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
