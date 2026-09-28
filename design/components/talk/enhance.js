// Voice mode: talk with the assistant.
//
// Start voice mode asks for the microphone once. Then, a turn at a time:
// listen until the person has spoken and stopped (or pressed Done
// speaking), write it down on the computer that hosts the workspace, send
// it as their message through the chat's own form, wait for the reply,
// read it aloud in a voice on this device, and listen again. The
// microphone is not recorded while a reply is read. After 30 seconds with
// nothing heard it pauses. End voice mode, or Escape anywhere, stops it
// all and lets the microphone go.
(function () {
  "use strict";
  var QUIET_END = 1200, NOTHING = 30000, LONGEST = 120000;

  function localVoice() {
    if (!window.speechSynthesis) return null;
    var all = speechSynthesis.getVoices().filter(function (v) { return v.localService; });
    var lang = (document.documentElement.lang || navigator.language || "en").slice(0, 2).toLowerCase();
    return all.filter(function (v) { return v.default && v.lang.slice(0, 2).toLowerCase() === lang; })[0] ||
      all.filter(function (v) { return v.lang.slice(0, 2).toLowerCase() === lang; })[0] || null;
  }
  // Long text is read a sentence or so at a time: some browsers stop a
  // single long reading part way.
  function pieces(text) {
    var out = [], parts = text.replace(/\s+/g, " ").match(/[^.!?]+[.!?]*\s*/g) || [text];
    var cur = "";
    parts.forEach(function (p) {
      if ((cur + p).length > 220 && cur) { out.push(cur.trim()); cur = ""; }
      cur += p;
    });
    if (cur.trim()) out.push(cur.trim());
    return out;
  }

  function arm(el) {
    if (el._armed || !window.swSpeech || !window.swSpeech.supported() || !(window.AudioContext || window.webkitAudioContext)) return;
    var ta = document.getElementById(el.getAttribute("data-message"));
    var form = ta && ta.form;
    if (!form) return;
    el._armed = true;
    el.hidden = false;
    var toggle = el.querySelector(".sw-talk__toggle"), done = el.querySelector(".sw-talk__done"), skip = el.querySelector(".sw-talk__skip");
    var said = el.querySelector(".sw-talk__state"), aloud = el.querySelector("input[type=checkbox]"), novoice = el.querySelector(".sw-talk__novoice");
    var stream = null, ctx = null, analyser = null, rec = null, chunks = [], timer = null, active = false, voiceWaited = false;
    try { if (localStorage.getItem("sw-talk-aloud") === "0") aloud.checked = false; } catch (e) {}
    aloud.addEventListener("change", function () { try { localStorage.setItem("sw-talk-aloud", aloud.checked ? "1" : "0"); } catch (e) {} });

    function state(s, words) {
      el.setAttribute("data-state", s);
      if (words !== undefined && said.textContent !== words) said.textContent = words;
      done.hidden = !(s === "listening" || s === "hearing");
      skip.hidden = s !== "speaking";
    }
    function level() {
      var buf = new Float32Array(analyser.fftSize);
      analyser.getFloatTimeDomainData(buf);
      var sum = 0;
      for (var i = 0; i < buf.length; i++) sum += buf[i] * buf[i];
      return Math.sqrt(sum / buf.length);
    }
    function listen() {
      if (!active) return;
      chunks = [];
      var type = "";
      ["audio/webm;codecs=opus", "audio/ogg;codecs=opus", "audio/mp4"].some(function (t) { if (MediaRecorder.isTypeSupported(t)) { type = t; return true; } return false; });
      rec = type ? new MediaRecorder(stream, { mimeType: type }) : new MediaRecorder(stream);
      rec.addEventListener("dataavailable", function (e) { if (e.data && e.data.size) chunks.push(e.data); });
      rec.start(250);
      state("listening", "Listening.");
      var began = Date.now(), floor = 0, samples = 0, speaking = false, spokeAt = 0, quietSince = 0;
      clearInterval(timer);
      timer = setInterval(function () {
        var now = Date.now(), v = level();
        if (now - began < 400) { floor += v; samples++; return; }
        var base = Math.max(0.004, floor / Math.max(1, samples));
        if (v > base * 3 && v > 0.012) {
          if (!speaking) { speaking = true; spokeAt = now; state("hearing", "Hearing you."); }
          quietSince = 0;
        } else if (speaking) {
          if (!quietSince) quietSince = now;
          if (now - quietSince > QUIET_END && now - spokeAt > 500) finish();
        }
        if (!speaking && now - began > NOTHING) pause("Paused: nothing was heard for 30 seconds. Press Continue voice mode to talk again.");
        if (now - began > LONGEST) finish();
      }, 60);
    }
    function stopRecorder() {
      clearInterval(timer);
      return new Promise(function (ok) {
        if (!rec || rec.state === "inactive") return ok(null);
        var r = rec;
        rec = null;
        r.addEventListener("stop", function () { ok(new Blob(chunks, { type: r.mimeType || "audio/webm" })); }, { once: true });
        r.stop();
      });
    }
    function finish() {
      if (!active) return;
      stopRecorder().then(function (blob) {
        if (!blob || !active) return;
        state("writing", "Writing down what you said.");
        return window.swSpeech.toWav(blob).then(function (wav) {
          return fetch(el.getAttribute("data-action"), { method: "POST", body: wav, headers: { "Content-Type": "audio/wav" }, credentials: "same-origin" });
        }).then(function (r) {
          return r.json().then(function (b) { if (!r.ok) throw new Error(b.error || "It could not be written down."); return (b.text || "").trim(); });
        }).then(function (text) {
          if (!active) return;
          if (!text) { state("listening", "No words were heard. Listening again."); setTimeout(listen, 600); return; }
          ta.value = text;
          ta.dispatchEvent(new Event("input", { bubbles: true }));
          state("answering", "Sent. The assistant is answering.");
          if (form.requestSubmit) form.requestSubmit(); else form.submit();
        });
      }).catch(function (e) { pause((e && e.message) || "Something went wrong, so voice mode paused."); });
    }
    function speak(text) {
      var voice = localVoice();
      if (!aloud.checked || !voice || !text) {
        novoice.hidden = !!voice || !aloud.checked;
        listen();
        return;
      }
      novoice.hidden = true;
      state("speaking", "Reading the reply.");
      var list = pieces(text), i = 0, quietFor = 0;
      // Some browsers now and then never say a reading has ended: when
      // nothing has been spoken for a while, it has.
      clearInterval(timer);
      timer = setInterval(function () {
        if (el.getAttribute("data-state") !== "speaking") { clearInterval(timer); return; }
        quietFor = speechSynthesis.speaking || speechSynthesis.pending ? 0 : quietFor + 500;
        if (quietFor >= 2000) { clearInterval(timer); speechSynthesis.cancel(); listen(); }
      }, 500);
      function next() {
        if (!active || el.getAttribute("data-state") !== "speaking") return;
        if (i >= list.length) { listen(); return; }
        var u = new SpeechSynthesisUtterance(list[i++]);
        u.voice = voice;
        u.lang = voice.lang;
        u.onend = next;
        u.onerror = next;
        speechSynthesis.speak(u);
      }
      next();
    }
    function pause(words) {
      stopRecorder();
      if (window.speechSynthesis) speechSynthesis.cancel();
      state("paused", words);
      toggle.textContent = "Continue voice mode";
      active = false;
    }
    function end() {
      active = false;
      stopRecorder();
      if (window.speechSynthesis) speechSynthesis.cancel();
      if (stream) stream.getTracks().forEach(function (t) { t.stop(); });
      if (ctx) ctx.close();
      stream = ctx = analyser = null;
      toggle.textContent = "Start voice mode";
      state("off", "Voice mode is off. The microphone is off.");
    }
    function start() {
      var go = function () {
        active = true;
        toggle.textContent = "End voice mode";
        listen();
      };
      if (stream) { go(); return; }
      state("off", "Asking for the microphone.");
      navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: true, noiseSuppression: true } }).then(function (s) {
        stream = s;
        var Ctx = window.AudioContext || window.webkitAudioContext;
        ctx = new Ctx();
        analyser = ctx.createAnalyser();
        analyser.fftSize = 1024;
        ctx.createMediaStreamSource(s).connect(analyser);
        if (!voiceWaited && window.speechSynthesis) { voiceWaited = true; speechSynthesis.getVoices(); }
        go();
      }, function (e) {
        state("off", e && e.name === "NotAllowedError"
          ? "The microphone was not allowed, so voice mode cannot listen. It can be allowed in the browser's settings for this site."
          : "The microphone could not be used, so voice mode cannot listen.");
      });
    }
    toggle.addEventListener("click", function () {
      if (active) end(); else start();
    });
    done.addEventListener("click", finish);
    skip.addEventListener("click", function () { if (window.speechSynthesis) speechSynthesis.cancel(); listen(); });
    document.addEventListener("keydown", function (e) { if (e.key === "Escape" && (active || stream)) end(); });
    document.addEventListener("sw:turn-done", function (e) {
      if (!active || el.getAttribute("data-state") !== "answering") return;
      var d = e.detail || {};
      var msg = d.id && document.getElementById("msg-" + d.id);
      var body = msg && msg.querySelector(".sw-message__body");
      speak(body ? body.innerText.trim() : "");
    });
  }
  function init() { document.querySelectorAll("[data-component=talk]").forEach(arm); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
