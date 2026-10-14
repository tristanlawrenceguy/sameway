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
    form.addEventListener("submit", function (event) { send(form, file, event); });
    file.addEventListener("invalid", function () {
      say(file.files && file.files[0] ? file.validationMessage : "Select a file to add");
    });
    check();
  }
  // A big file says how much of it has gone: past ten seconds a wait wants
  // a measure, not "Adding the file…" (NN/g, progress indicators). The
  // form is sent as the browser would send it, and where the server sends
  // the person is where they go, with what it said there; the page asked
  // for on the way is the page's own fetch (sameway-live), so what was
  // said waits for them. The bar shows every step; the words change each
  // quarter, so a screen reader hears now and then, not all the time
  // (WCAG 4.1.3). A small file goes as a form always went.
  var BIG = 10 * 1024 * 1024;
  function size(n) {
    return n >= 1073741824 ? (n / 1073741824).toFixed(1) + " GB" : Math.max(1, Math.round(n / 1048576)) + " MB";
  }
  function send(form, file, event) {
    var f = file.files && file.files[0];
    var status = document.getElementById(form.getAttribute("data-busy-target"));
    if (event.defaultPrevented || !f || f.size < BIG || !window.FormData || !status) return;
    event.preventDefault();
    var bar = form.querySelector(".sw-upload__progress");
    if (!bar) {
      bar = document.createElement("progress");
      bar.className = "sw-upload__progress";
      bar.max = 100;
      bar.setAttribute("aria-label", "Sent so far");
      status.parentNode.insertBefore(bar, status);
    }
    bar.hidden = false;
    bar.value = 0;
    var quarter = -1;
    var xhr = new XMLHttpRequest();
    xhr.open("POST", form.action);
    xhr.setRequestHeader("X-Requested-With", "sameway-live");
    xhr.upload.onprogress = function (e) {
      if (!e.lengthComputable) return;
      var part = e.loaded / e.total;
      bar.value = Math.round(part * 100);
      if (Math.floor(part * 4) !== quarter) {
        quarter = Math.floor(part * 4);
        sw.status(status, "working", "Sending the file: " + size(e.loaded) + " of " + size(e.total) + " sent.");
      }
    };
    xhr.upload.onload = function () {
      bar.value = 100;
      sw.status(status, "working", "Sent. Reading the file…");
    };
    xhr.onload = function () { location.assign(xhr.responseURL || location.href); };
    xhr.onerror = xhr.onabort = function () {
      bar.hidden = true;
      form.removeAttribute("aria-busy");
      if (form.closest("[data-region]")) form.closest("[data-region]").setAttribute("data-state", "idle");
      form.querySelectorAll("[aria-disabled]").forEach(function (b) { b.removeAttribute("aria-disabled"); });
      document.title = document.title.replace(/^⏳ /, "");
      sw.status(status, "failed", "The file could not be sent: the connection was lost. Try again.");
    };
    xhr.send(new FormData(form));
  }
  sw.arm("[data-component=upload]", arm);
})();
