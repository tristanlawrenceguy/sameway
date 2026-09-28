// A recording's player, with its words in sight.
//
// The browser's own player is there without this script; it plays, but
// its controls differ from browser to browser, some cannot be reached by
// keyboard, and some are too faint to see. This puts a bar in its place:
// Play or Pause, Back 15 seconds, Forward 30 seconds, a speed, and a seek
// bar that says where it is in words. The transcript's times play from
// their line, and the line being heard is marked, never announced and
// never scrolled to. Nothing plays by itself.
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
    var media = fig.querySelector("audio, video");
    if (!media) return;
    fig._armed = true;
    var title = (fig.querySelector(".sw-media__title") || {}).textContent || "the recording";
    var bar = document.createElement("div");
    bar.className = "sw-media__bar";
    bar.setAttribute("role", "group");
    bar.setAttribute("aria-label", "Player: " + title);
    var play = button("Play", " " + title);
    var back = button("Back 15 seconds");
    var fwd = button("Forward 30 seconds");
    var time = document.createElement("span");
    time.className = "sw-media__time";
    time.setAttribute("aria-hidden", "true");
    var seek = document.createElement("input");
    seek.type = "range";
    seek.className = "sw-media__seek";
    seek.min = "0";
    seek.step = "1";
    seek.value = "0";
    seek.setAttribute("aria-label", "Where in " + title);
    var speed = document.createElement("label");
    speed.className = "sw-media__speed";
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
    media.removeAttribute("controls");
    // A recording that is only heard has nothing to see; a video stays.
    if (media.tagName === "AUDIO") media.hidden = true;
    media.insertAdjacentElement("afterend", bar);

    var focused = false;
    function said() {
      var d = isFinite(media.duration) ? media.duration : 0;
      seek.setAttribute("aria-valuetext", words(media.currentTime) + (d ? " of " + words(d) : ""));
    }
    function show() {
      var d = isFinite(media.duration) ? media.duration : 0;
      time.textContent = clock(media.currentTime) + (d ? " / " + clock(d) : "");
      if (d) seek.max = String(Math.floor(d));
      seek.value = String(Math.floor(media.currentTime));
      // Where it is is said when someone moves it, not on every tick.
      if (!focused) said();
      mark();
    }
    function state() {
      play.firstChild.nodeValue = media.paused ? "Play" : "Pause";
      fig.setAttribute("data-state", media.paused ? "paused" : "playing");
    }
    var cues = Array.prototype.slice.call(fig.querySelectorAll(".sw-media__cue"));
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
    play.addEventListener("click", function () { if (media.paused) media.play(); else media.pause(); });
    back.addEventListener("click", function () { to(media.currentTime - 15); });
    fwd.addEventListener("click", function () { to(media.currentTime + 30); });
    rate.addEventListener("change", function () { media.playbackRate = parseFloat(rate.value) || 1; });
    seek.addEventListener("input", function () { to(parseFloat(seek.value) || 0); said(); });
    seek.addEventListener("focus", function () { focused = true; said(); });
    seek.addEventListener("blur", function () { focused = false; });
    seek.addEventListener("keydown", function (e) {
      var step = { ArrowLeft: -5, ArrowDown: -5, ArrowRight: 5, ArrowUp: 5, PageDown: -60, PageUp: 60 }[e.key];
      if (step) { e.preventDefault(); to(media.currentTime + step); said(); }
      else if (e.key === "Home") { e.preventDefault(); to(0); said(); }
      else if (e.key === "End" && isFinite(media.duration)) { e.preventDefault(); to(media.duration); said(); }
    });
    ["timeupdate", "loadedmetadata", "durationchange", "seeked"].forEach(function (ev) { media.addEventListener(ev, show); });
    ["play", "pause", "ended"].forEach(function (ev) { media.addEventListener(ev, function () { state(); mark(); }); });
    fig.querySelectorAll(".sw-media__at").forEach(function (a) {
      a.addEventListener("click", function (e) {
        e.preventDefault();
        var c = a.closest(".sw-media__cue");
        to(parseFloat(c.getAttribute("data-start")) || 0);
        media.play();
      });
    });
    state();
    show();
  }
  function armMake(fig) {
    var form = fig.querySelector(".sw-media__make"), media = fig.querySelector("audio source, video source") || fig.querySelector("audio, video");
    if (!form || form._armed || !media) return;
    form._armed = true;
    var said = form.querySelector(".sw-media__making"), btn = form.querySelector("button");
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
    var key = "sw-media-auto:" + form.action;
    var tried = false;
    try { tried = sessionStorage.getItem(key) === "1"; sessionStorage.setItem(key, "1"); } catch (e) {}
    if (form.hasAttribute("data-auto") && !tried) go();
  }
  function init(root) { (root || document).querySelectorAll("[data-component=media]").forEach(function (f) { arm(f); armMake(f); }); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", function () { init(); });
  else init();
  document.addEventListener("sw:refresh", function () { init(); });
})();
