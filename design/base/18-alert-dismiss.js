(function () {
  "use strict";
  var btns = document.querySelectorAll("[data-dismiss]");
  for (var i = 0; i < btns.length; i++) {
    btns[i].addEventListener("click", function () {
      var alert = this.closest(".sw-alert");
      if (alert) alert.remove();
    });
  }
})();
