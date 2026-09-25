// What the picture shows is asked only once a picture is chosen.
//
// Without scripts the field is always there, and says to leave it empty
// for other files. With them it waits until the file chosen is a picture,
// so a PDF is one field and one button.
(function () {
  "use strict";
  function arm(form) {
    var file = form.querySelector(".sw-upload__field");
    var about = form.querySelector(".sw-upload__about");
    if (!file || !about || form._armed) return;
    form._armed = true;
    function show() {
      var f = file.files && file.files[0];
      about.hidden = !(f && /^image\//.test(f.type));
    }
    file.addEventListener("change", show);
    show();
  }
  function init() { document.querySelectorAll("[data-component=upload]").forEach(arm); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
