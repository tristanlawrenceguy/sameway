// A recording's player, with its words in sight.
//
// The browser's own player is there without this script; it plays, but
// its controls differ from browser to browser, some cannot be reached by
// keyboard, and some are too faint to see. This puts a bar in its place:
// Play or Pause, Back 15 seconds, Forward 30 seconds, a speed, and a seek
// bar that says where it is in words. The transcript's times play from
// their line, and the line being heard is marked, never announced and
// never scrolled to. Nothing plays by itself, one recording pauses when
// another starts, and one this browser cannot play says so.
(function () {
  "use strict";
  function clock(s) {
    s = Math.max(0, Math.floor(s || 0));
    var h = Math.floor(s / 3600), m = Math.floor(s / 60) % 60, x = s % 60;
    var mm = h ? String(m).padStart(2, "0") : String(m);
    return (h ? h + ":" : "") + mm + ":" + String(x).padStart(2, "0");
  }
  function words(s) {
    s = Math.max(0, Math.floor(s || 0));
    var h = Math.floor(s / 3600), m = Math.floor(s / 60) % 60, x = s % 60, out = [];
    if (h) out.push(h + (h === 1 ? " hour" : " hours"));
    if (m) out.push(m + (m === 1 ? " minute" : " minutes"));
    if (x || !out.length) out.push(x + (x === 1 ? " second" : " seconds"));
    return out.join(" ");
  }
  function button(label, hidden) {
    var b = document.createElement("button");
    b.type = "button";
    b.className = "sw-button sw-button--secondary sw-pressable";
    b.textContent = label;
    if (hidden) {
      var span = document.createElement("span");
      span.className = "sw-visually-hidden";
      span.textContent = hidden;
      b.appendChild(span);
    }
    return b;
  }
  function arm(fig) {
    if (fig._armed) return;
    var media = fig.querySelector("audio");
    if (!media) return;
    fig._armed = true;
    var title = (fig.querySelector(".sw-audio__title") || {}).textContent || "the recording";
    var bar = document.createElement("div");
    bar.className = "sw-audio__bar";
    var play = button("Play", " " + title);
    var back = button("Back 15 seconds");
    var fwd = button("Forward 30 seconds");
    var time = document.createElement("span");
    time.className = "sw-audio__time";
    time.setAttribute("aria-hidden", "true");
    var seek = document.createElement("input");
    seek.type = "range";
    seek.className = "sw-audio__seek";
    seek.min = "0";
    seek.step = "1";
    seek.value = "0";
    seek.setAttribute("aria-label", "Where in " + title);
    var speed = document.createElement("label");
    speed.className = "sw-audio__speed";
    speed.textContent = "Speed ";
    var rate = document.createElement("select");
    [["0.75", "0.75 times"], ["1", "Normal"], ["1.25", "1.25 times"], ["1.5", "1.5 times"], ["2", "Twice"]].forEach(function (o) {
      var opt = document.createElement("option");
      opt.value = o[0];
      opt.textContent = o[1];
      if (o[0] === "1") opt.selected = true;
      rate.appendChild(opt);
    });
    speed.appendChild(rate);
    bar.append(play, back, fwd, time, seek, speed);
    // Empty until this browser cannot play it; then it says so, once.
    var trouble = document.createElement("p");
    trouble.className = "sw-audio__trouble";
    trouble.setAttribute("role", "status");
    media.removeAttribute("controls");
    media.hidden = true;
    media.insertAdjacentElement("afterend", bar);
    bar.insertAdjacentElement("afterend", trouble);

    var focused = false;
    function said() {
      var d = isFinite(media.duration) ? media.duration : 0;
      seek.setAttribute("aria-valuetext", words(media.currentTime) + (d ? " of " + words(d) : ""));
    }
    function show() {
      var d = isFinite(media.duration) ? media.duration : 0;
      time.textContent = clock(media.currentTime) + (d ? " / " + clock(d) : "");
      if (d) seek.max = String(Math.floor(d));
      // While it has focus the seek bar moves only when someone moves it:
      // a value that changes on every tick is read out on every tick.
      if (!focused) { seek.value = String(Math.floor(media.currentTime)); said(); }
      mark();
    }
    function state() {
      play.firstChild.nodeValue = media.paused ? "Play" : "Pause";
      fig.setAttribute("data-state", media.paused ? "paused" : "playing");
    }
    var cues = Array.prototype.slice.call(fig.querySelectorAll(".sw-audio__cue"));
    function mark() {
      var t = media.currentTime, now = null;
      cues.forEach(function (c) { if (parseFloat(c.getAttribute("data-start")) <= t + 0.05) now = c; });
      cues.forEach(function (c) {
        if (c === now && !media.paused) c.setAttribute("aria-current", "true");
        else c.removeAttribute("aria-current");
      });
    }
    function to(t) {
      var d = isFinite(media.duration) ? media.duration : Infinity;
      media.currentTime = Math.min(Math.max(0, t), d);
    }
    // Moved by keyboard: the bar and its words follow at once.
    function moved(t) {
      to(t);
      seek.value = String(Math.floor(media.currentTime));
      said();
    }
    function cannot() {
      bar.hidden = true;
      fig.removeAttribute("data-state");
      trouble.textContent = "This browser cannot play " + title + ". Download it to play it elsewhere.";
    }
    function start() {
      var p = media.play();
      if (p && p.catch) p.catch(function (e) { if (e && e.name === "NotSupportedError") cannot(); });
    }
    // A failed source says so on itself; the recording on the audio element.
    media.addEventListener("error", cannot);
    var sources = media.querySelectorAll("source");
    if (sources.length) sources[sources.length - 1].addEventListener("error", cannot);
    if (media.error || media.networkState === 3) cannot();
    play.addEventListener("click", function () { if (media.paused) start(); else media.pause(); });
    back.addEventListener("click", function () { to(media.currentTime - 15); });
    fwd.addEventListener("click", function () { to(media.currentTime + 30); });
    rate.addEventListener("change", function () { media.playbackRate = parseFloat(rate.value) || 1; });
    seek.addEventListener("input", function () { to(parseFloat(seek.value) || 0); said(); });
    seek.addEventListener("focus", function () { focused = true; said(); });
    seek.addEventListener("blur", function () { focused = false; });
    seek.addEventListener("keydown", function (e) {
      var step = { ArrowLeft: -5, ArrowDown: -5, ArrowRight: 5, ArrowUp: 5, PageDown: -60, PageUp: 60 }[e.key];
      if (step) { e.preventDefault(); moved(media.currentTime + step); }
      else if (e.key === "Home") { e.preventDefault(); moved(0); }
      else if (e.key === "End" && isFinite(media.duration)) { e.preventDefault(); moved(media.duration); }
    });
    ["timeupdate", "loadedmetadata", "durationchange", "seeked"].forEach(function (ev) { media.addEventListener(ev, show); });
    ["play", "pause", "ended"].forEach(function (ev) { media.addEventListener(ev, function () { state(); mark(); }); });
    // One sound at a time: starting this one pauses any other on the page.
    media.addEventListener("play", function () {
      document.querySelectorAll("[data-component=audio] audio").forEach(function (a) { if (a !== media && !a.paused) a.pause(); });
    });
    fig.querySelectorAll(".sw-audio__at").forEach(function (a) {
      a.addEventListener("click", function (e) {
        e.preventDefault();
        var c = a.closest(".sw-audio__cue");
        to(parseFloat(c.getAttribute("data-start")) || 0);
        start();
      });
    });
    state();
    show();
  }
  function armMake(fig) {
    var form = fig.querySelector(".sw-audio__make"), media = fig.querySelector("audio source") || fig.querySelector("audio");
    if (!form || form._armed || !media) return;
    form._armed = true;
    var said = form.querySelector(".sw-audio__making"), btn = form.querySelector("button");
    var src = media.getAttribute("src");
    function go() {
      btn.setAttribute("aria-disabled", "true");
      said.textContent = "Reading the recording on this device.";
      window.swSpeech.toWav(src).then(function (wav) {
        said.textContent = "Sending it to be written down.";
        return fetch(form.action, { method: "POST", body: wav, headers: { "Content-Type": "audio/wav" }, credentials: "same-origin" });
      }).then(function (r) { location.href = r.url; }, function () {
        btn.removeAttribute("aria-disabled");
        said.textContent = "This browser could not read the recording, so it was not written down.";
      });
    }
    form.addEventListener("submit", function (e) {
      e.preventDefault();
      if (btn.getAttribute("aria-disabled") !== "true") go();
    });
    // Once per recording per visit, so a failure does not repeat itself.
    var key = "sw-audio-auto:" + form.action;
    var tried = false;
    try { tried = sessionStorage.getItem(key) === "1"; sessionStorage.setItem(key, "1"); } catch (e) {}
    if (form.hasAttribute("data-auto") && !tried) go();
  }
  function init(root) { (root || document).querySelectorAll("[data-component=audio]").forEach(function (f) { arm(f); armMake(f); }); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", function () { init(); });
  else init();
  document.addEventListener("sw:refresh", function () { init(); });
})();
