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
  function composeKeyHandler() {
    if (window.matchMedia && window.matchMedia("(pointer: coarse)").matches) return;
    document.querySelectorAll("form.sw-compose textarea").forEach(function (textarea) {
      if (textarea._composeHandled) return;
      textarea._composeHandled = true;
      var form = textarea.closest("form.sw-compose");
      var hint = document.getElementById(textarea.id + "-hint");
      if (hint && hint.textContent.indexOf("Enter sends") < 0) hint.textContent += " Enter sends; Shift+Enter starts a new line.";
      textarea.addEventListener("keydown", function (e) {
        if (e.key === "Enter" && !e.shiftKey && !e.isComposing) {
          e.preventDefault();
          if (form.requestSubmit) form.requestSubmit();
          else form.submit();
        }
      });
    });
  }

  function init() { composeKeyHandler(); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  // Parts refreshed during a live turn are armed too; armed ones say so.
  document.addEventListener("sw:refresh", init);
})();
