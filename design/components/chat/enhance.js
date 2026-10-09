// The chat, with its scripts: shaped by the person, and Enter sends. (The
// turn as it happens, shown in it, is design/base: it reaches the canvas.)
//
// Pop out opens the conversation in a small window of its own, so it can
// sit beside whatever else is on the screen; without scripts the link
// opens a tab, which is the same conversation. The log can be dragged
// taller or shorter (that is CSS), and the height a person settles on is
// kept in this browser so it is the same next time.
(function () {
  "use strict";
  var KEY = "sameway:chat-height";

  function popout(a) {
    a.addEventListener("click", function (e) {
      if (!window.open) return;
      var win = window.open(a.getAttribute("href"), "sameway-chat", "popup=yes,width=460,height=760");
      if (win) e.preventDefault();
    });
  }

  // The log opens at its end, where the newest message is, unless the
  // address names a message to land on.
  function toEnd(log) {
    if (/^#msg-/.test(location.hash)) return;
    var behaviour = log.style.scrollBehavior;
    log.style.scrollBehavior = "auto";
    log.scrollTop = log.scrollHeight;
    log.style.scrollBehavior = behaviour;
  }

  function remember(log) {
    var saved = null;
    try { saved = localStorage.getItem(KEY); } catch (err) { saved = null; }
    if (saved && log.querySelector("li")) log.style.height = saved;
    toEnd(log);
    if (!window.ResizeObserver) return;
    var first = true;
    new ResizeObserver(function () {
      // The first call reports the size as laid out, not a choice.
      if (first) { first = false; return; }
      if (!log.style.height) return;
      try { localStorage.setItem(KEY, log.style.height); } catch (err) { /* private windows keep nothing */ }
    }).observe(log);
  }

  function init() {
    document.querySelectorAll("a[data-popout]").forEach(popout);
    document.querySelectorAll(".sw-chat__log").forEach(remember);
  }
  sw.ready(init);
})();

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
