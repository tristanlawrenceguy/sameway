// Each upload form, on its own, wherever it is: the list's or a block's.
//
// What the picture shows is asked only once a picture is chosen: without
// scripts the field is always there, and says to leave it empty for other
// files. A form sent with no file, or one too big, says so above the field
// in words, and the field is marked as having a problem; a file too big is
// caught before it is sent, not after 4 GB has gone. While the file goes
// its status says so (the status component).
(function () {
  "use strict";
  function arm(form) {
    var file = form.querySelector(".sw-upload__field");
    var about = form.querySelector(".sw-upload__about");
    var err = form.querySelector(".sw-upload__error");
    if (!file || form._armed) return;
    form._armed = true;
    var max = Number(file.getAttribute("data-max")) || 0;
    function say(words) {
      if (err) err.innerHTML = words ? '<span class="sw-visually-hidden">Error: </span>' : "";
      if (err && words) err.appendChild(document.createTextNode(words));
      if (words) file.setAttribute("aria-invalid", "true"); else file.removeAttribute("aria-invalid");
    }
    function check() {
      var f = file.files && file.files[0];
      if (about) about.hidden = !(f && /^image\//.test(f.type));
      if (f && max && f.size > max) {
        file.setCustomValidity("The selected file must be smaller than 4 GB");
      } else if (f && f.size === 0) {
        file.setCustomValidity("The selected file is empty");
      } else {
        file.setCustomValidity("");
      }
      say(file.validity.valid ? "" : file.validationMessage && f ? file.validationMessage : "");
    }
    file.addEventListener("change", check);
    file.addEventListener("invalid", function () {
      say(file.files && file.files[0] ? file.validationMessage : "Select a file to add");
    });
    check();
  }
  sw.arm("[data-component=upload]", arm);
})();
