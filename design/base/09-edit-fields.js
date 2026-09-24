// The whole record, edited where it is.
//
// A record's page says the least it can at rest, so what it shows is not
// what can be changed: the title is the heading, a few fields are chips,
// an empty field is not there at all. The page carries what can be
// edited in a template it does not show (editfields.go): every field of
// the type in its order, marked the way 08-edit.js reads a field. Edit
// builds the form from that, so a person can rename a record, set its
// date, choose its project or fill in a field it never had, by hand, in
// place, without asking.
(function () {
  "use strict";

  // swEditFields is what the block says can be edited, or null when it
  // says nothing and the marked elements on it are what there is.
  window.swEditFields = function (block) {
    var t = block.querySelector("template[data-edit-fields]");
    if (!t || !t.content || !t.content.children.length) return null;
    return Array.prototype.slice.call(t.content.children);
  };

  // A record just made by hand arrives at #edit, open in its editor, its
  // name ready to change; the address is put back, so a reload does not
  // open it again.
  function openNew() {
    if (location.hash !== "#edit") return;
    if (window.history && history.replaceState) history.replaceState(null, "", location.pathname + location.search);
    setTimeout(function () {
      var edit = document.querySelector("[data-block-id] [data-edit]");
      if (edit) edit.click();
      var name = document.querySelector(".sw-inline-form input[type=text]");
      if (name && name.select) name.select();
    }, 0);
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", openNew); else openNew();
})();
