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
})();
