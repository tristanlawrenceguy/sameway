// Enter sends a chat message, Shift+Enter starts a new line.
(function () {
  "use strict";

  // Enter in the chat composer sends, the way the Send button does: through
  // the form's own submit, so the required check, the busy state and the
  // double-send guard all see it. Shift+Enter starts a new line.
  //
  // Not on a touch screen: its keyboard has no Shift+Enter, so Enter has to
  // make a new line there, and Send sends. The hint says Enter sends only
  // where it does, so it is never untrue: without this script, or on a
  // phone, it says nothing about Enter.
  if (window.matchMedia && window.matchMedia("(pointer: coarse)").matches) return;
  sw.arm("form.sw-compose textarea", function (textarea) {
    var form = textarea.closest("form.sw-compose");
    var hint = document.getElementById(textarea.id + "-hint");
    if (hint && hint.textContent.indexOf("Enter sends") < 0) hint.textContent += " Enter sends; Shift+Enter starts a new line.";
    textarea.addEventListener("keydown", function (e) {
      // Not the Enter that finishes a Japanese, Chinese or Korean word,
      // which Safari sends as 229 after the word is done; and nothing is
      // sent from an empty box.
      if (e.key === "Enter" && !e.shiftKey && !e.isComposing && e.keyCode !== 229) {
        e.preventDefault();
        if (!textarea.value.trim()) return;
        if (form.requestSubmit) form.requestSubmit();
        else form.submit();
      }
    });
  });
})();
