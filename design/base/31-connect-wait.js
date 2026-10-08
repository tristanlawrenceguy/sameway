// The connect card follows what the person is doing about it.
//
// A person told to install Ollama installed it, came back, and had to know
// to press Check again; a model being fetched said how far it had come only
// when asked. Now, while the card is up, the page asks the server every few
// seconds what the card would say (GET /model/wait, connect.go) and, when
// that changes, follows (17-refresh.js): Ollama found, a free model offered,
// the fetch's progress, the assistant ready. Not while a key is being
// typed into the card, which a fresh card would empty. Check again stays,
// for a page without scripts.
(function () {
  "use strict";
  if (!window.fetch) return;
  var EVERY = 4000;
  var timer = null;

  function typing(card) {
    var fields = card.querySelectorAll("input:not([type=hidden]), textarea");
    for (var i = 0; i < fields.length; i++) if (fields[i].value) return true;
    return false;
  }

  function look() {
    timer = null;
    var card = document.querySelector(".sw-connect[data-wait]");
    if (!card) return;
    fetch("/model/wait", { credentials: "same-origin", cache: "no-store" })
      .then(function (r) { if (!r.ok) throw new Error("the server answered " + r.status); return r.text(); })
      .then(function (now) {
        if (now !== card.getAttribute("data-wait") && !typing(card)) sw.refresh(0);
        else wait();
      })
      // A server that cannot say (another person's page, a server gone) is
      // left alone: Check again still works.
      .catch(function () {});
  }

  function wait() {
    clearTimeout(timer);
    timer = setTimeout(look, EVERY);
  }

  wait();
  document.addEventListener("sw:refresh", wait);
})();
