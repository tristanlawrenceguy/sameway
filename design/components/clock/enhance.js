// Progressive enhancement for the clock.
//
// Without scripts the clock is the time the page was made, the forms set
// a timer or an alarm, and a reminder that has rung shows the next time
// the page loads. With scripts the time keeps moving, and a reminder
// rings the moment its time comes while the page is open: it appears in
// the clock, says so to a screen reader, sounds unless the clock is
// quiet, and shows a notification when the browser has been allowed to.
// It stays until Dismiss or five more minutes.
(function () {
  "use strict";
  function pad(n) { return (n < 10 ? "0" : "") + n; }

  function tick(clock) {
    var time = clock.querySelector(".sw-clock__time");
    var base = Date.parse(clock.getAttribute("data-now"));
    if (!time || isNaN(base)) return;
    var start = Date.now();
    function show() {
      var d = new Date(base + (Date.now() - start));
      time.textContent = pad(d.getHours()) + ":" + pad(d.getMinutes());
      time.setAttribute("datetime", d.toISOString());
    }
    show();
    setInterval(show, 5000);
  }

  // One sound for the page, woken by the person's first press or key:
  // a browser keeps sound made without one silent, and a reminder rings
  // long after any press. Until then a ring still shows and notifies.
  var audio = null;
  function sound() {
    if (!audio) {
      var AC = window.AudioContext || window.webkitAudioContext;
      if (!AC) return null;
      try { audio = new AC(); } catch (e) { return null; }
    }
    if (audio.state === "suspended" && audio.resume) audio.resume().catch(function () {});
    return audio;
  }
  ["pointerdown", "keydown"].forEach(function (kind) {
    document.addEventListener(kind, function () { if (document.querySelector(".sw-clock")) sound(); }, { once: true, capture: true });
  });

  // Three short notes, twice: enough to be heard, not enough to be a siren.
  function chime() {
    var ctx = sound();
    if (!ctx) return;
    try {
      [0, 0.2, 0.4, 1.2, 1.4, 1.6].forEach(function (at) {
        var o = ctx.createOscillator(), g = ctx.createGain();
        o.type = "sine"; o.frequency.value = 880;
        g.gain.setValueAtTime(0.0001, ctx.currentTime + at);
        g.gain.exponentialRampToValueAtTime(0.2, ctx.currentTime + at + 0.02);
        g.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + at + 0.15);
        o.connect(g); g.connect(ctx.destination);
        o.start(ctx.currentTime + at); o.stop(ctx.currentTime + at + 0.16);
      });
    } catch (e) { /* no sound is not a failure */ }
  }

  function el(html) {
    var t = document.createElement("template");
    t.innerHTML = html.trim();
    return t.content.firstElementChild;
  }
  function escape(s) {
    return String(s).replace(/[&<>"]/g, function (c) { return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]; });
  }

  // A reminder shows in every clock on the page, and is said once, as its
  // name, by the first: the list itself is not read out whole, so the
  // title is heard once, not with each button after it.
  function ring(d) {
    var rang = false;
    var said = document.querySelector(".sw-clock .sw-clock__said");
    // What it is about, when it is about something: Water, 3 of 8 glasses
    // so far. The fixed words said for a reminder about nothing are left.
    var about = d.text && d.text !== "It is time." && d.text !== d.title ? d.text : "";
    if (said) said.textContent = "Reminder: " + d.title + (about ? ". " + about : "");
    document.querySelectorAll(".sw-clock").forEach(function (clock) {
      var list = clock.querySelector(".sw-clock__ringing");
      if (!list || list.querySelector('[data-id="' + d.id + '"]')) return;
      var what = (d.href ? '<a class="sw-link sw-link--fill" href="' + escape(d.href) + '">' + escape(d.title) + "</a>" : escape(d.title)) +
        (about ? '<span class="sw-clock__about">' + escape(about) + "</span>" : "");
      list.appendChild(el('<div class="sw-clock__ring" data-id="' + escape(d.id) + '"><span class="sw-clock__bell" aria-hidden="true"></span><span class="sw-clock__what">' + what + "</span>" +
        '<form method="post" action="/clock/' + escape(d.id) + '/done"><button type="submit" class="sw-button sw-button--primary sw-pressable">Dismiss<span class="sw-visually-hidden"> ' + escape(d.title) + "</span></button></form>" +
        '<form method="post" action="/clock/' + escape(d.id) + '/snooze"><button type="submit" class="sw-button sw-button--quiet sw-pressable">5 more minutes<span class="sw-visually-hidden"> for ' + escape(d.title) + "</span></button></form></div>"));
      var item = clock.querySelector('.sw-clock__item[data-id="' + d.id + '"]');
      if (item) item.remove();
      if (clock.getAttribute("data-sound") !== "off") rang = true;
    });
    if (rang) chime();
    if (!/^⏰ /.test(document.title)) document.title = "⏰ " + document.title;
    if (window.Notification && Notification.permission === "granted") {
      // It stays until answered where the browser allows, as the ring on
      // the page does, and a press on it leads to what it is about.
      try {
        var n = new Notification(d.title, { body: d.text || "It is time.", tag: "sameway-" + d.id, requireInteraction: true });
        n.onclick = function () { window.focus(); if (d.url) location.href = d.url; n.close(); };
      } catch (e) { /* fine */ }
    }
  }

  function listen() {
    if (!window.EventSource) return;
    var es = new EventSource("/clock/stream");
    es.addEventListener("ring", function (e) {
      var d = {};
      try { d = JSON.parse(e.data); } catch (err) { return; }
      if (d.id) ring(d);
    });
  }

  // The way to allow notifications, only where they are possible and not
  // yet decided; a button that would do nothing is not shown.
  function askToNotify(clock) {
    if (clock.getAttribute("data-detail") === "glance" || !window.Notification || Notification.permission !== "default") return;
    var btn = el('<button type="button" class="sw-button sw-button--quiet sw-pressable sw-clock__notify">Allow notifications</button>');
    btn.addEventListener("click", function () {
      Notification.requestPermission().then(function () { btn.remove(); });
    });
    clock.appendChild(btn);
  }

  var listening = false;
  function init() {
    var clocks = document.querySelectorAll(".sw-clock");
    if (!clocks.length) return;
    clocks.forEach(function (c) {
      if (c._ticking) return;
      c._ticking = true;
      tick(c);
      askToNotify(c);
    });
    if (!listening) { listening = true; listen(); }
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init); else init();
  // A page that refreshed part of itself may hold a new clock.
  document.addEventListener("sw:refresh", init);
})();
