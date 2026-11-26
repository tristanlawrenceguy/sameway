// The chat, with its scripts. (The turn as it happens is design/base: it
// reaches the canvas.) Without them the pop-out link opens a tab, which is
// the same conversation, and Send sends.
(function () {
  "use strict";
  var KEY = "sameway:chat-height";

  // Pop out: a small window of its own, beside whatever else is on screen.
  function popout(a) {
    a.addEventListener("click", function (e) {
      if (window.open && window.open(a.getAttribute("href"), "sameway-chat", "popup=yes,width=460,height=760")) e.preventDefault();
    });
  }

  // The log opens at its end, where the newest message is, unless the
  // address names a message to land on. The height a person drags it to
  // (that is CSS) is kept in this browser.
  function log(el) {
    var saved = null;
    try { saved = localStorage.getItem(KEY); } catch (err) { /* private windows keep nothing */ }
    if (saved && el.querySelector("li")) el.style.height = saved;
    if (!/^#msg-/.test(location.hash)) {
      var behaviour = el.style.scrollBehavior;
      el.style.scrollBehavior = "auto";
      el.scrollTop = el.scrollHeight;
      el.style.scrollBehavior = behaviour;
    }
    if (!window.ResizeObserver) return;
    var first = true;
    new ResizeObserver(function () {
      // The first call reports the size as laid out, not a choice.
      if (first) { first = false; return; }
      if (!el.style.height) return;
      try { localStorage.setItem(KEY, el.style.height); } catch (err) { /* nothing kept */ }
    }).observe(el);
  }

  sw.ready(function () {
    document.querySelectorAll("a[data-popout]").forEach(popout);
    document.querySelectorAll(".sw-chat__log").forEach(log);
  });

  // Enter sends, the way the Send button does: through the form's own
  // submit, so the required check, the busy state and the double-send
  // guard all see it. Shift+Enter starts a new line.
  //
  // Not on a touch screen: its keyboard has no Shift+Enter, so Enter makes
  // a new line there, and Send sends. The hint says Enter sends only where
  // it does, so it is never untrue.
  if (window.matchMedia && window.matchMedia("(pointer: coarse)").matches) return;
  sw.arm("form.sw-compose textarea", function (textarea) {
    var form = textarea.closest("form.sw-compose");
    var hint = document.getElementById(textarea.id + "-hint");
    if (hint && hint.textContent.indexOf("Enter sends") < 0) hint.textContent += " Enter sends; Shift+Enter starts a new line.";
    textarea.addEventListener("keydown", function (e) {
      // Not the Enter that finishes a Japanese, Chinese or Korean word,
      // which Safari sends as 229 after the word is done; and nothing is
      // sent from an empty box.
      if (e.key !== "Enter" || e.shiftKey || e.isComposing || e.keyCode === 229) return;
      e.preventDefault();
      if (!textarea.value.trim()) return;
      if (form.requestSubmit) form.requestSubmit(); else form.submit();
    });
  });
})();
