// Voice mode: talk with the assistant.
//
// Start voice mode asks for the microphone. Then, a turn at a time:
// listen until the person has spoken and stopped (or pressed Done
// speaking), write it down on the computer that hosts the workspace, send
// it as their message through the chat's own form, wait for the reply,
// read it aloud in a voice on this device, and listen again. A reply that
// is not read aloud is read by the person, or by their screen reader, so
// then the microphone is let go and it waits for Speak. The microphone is
// not recorded while a reply is read. After 30 seconds with nothing heard
// it pauses and lets the microphone go. End voice mode, or Escape
// anywhere, stops it all.
(function () {
  "use strict";
  var QUIET_END = 1200, NOTHING = 30000, LONGEST = 120000;
  var TURN = { listening: "Done speaking", hearing: "Done speaking", writing: "Done speaking", answering: "Done speaking", speaking: "Skip the reply", waiting: "Speak" };

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
    if (el._armed || !sw.speech || !sw.speech.supported() || !(window.AudioContext || window.webkitAudioContext)) return;
    var ta = document.getElementById(el.getAttribute("data-message"));
    var form = ta && ta.form;
    if (!form) return;
    el._armed = true;
    el.hidden = false;
    var toggle = el.querySelector(".sw-talk__toggle"), turn = el.querySelector(".sw-talk__turn");
    var said = el.querySelector(".sw-talk__state"), aloud = el.querySelector("input[type=checkbox]"), novoice = el.querySelector(".sw-talk__novoice");
    var stream = null, ctx = null, analyser = null, rec = null, chunks = [], timer = null, active = false, voiceWaited = false;
    // A page can hold two chats (a canvas's and the side one), each with
    // voice mode; the second checkbox gets its own id, or its label would
    // press the first and it would be a 28px target.
    var n = 1, base = aloud.id, lab = aloud.labels && aloud.labels[0];
    if (!lab) {
      while (document.getElementById(base + "-" + n)) n++;
      lab = el.querySelector('label[for="' + base + '"]');
      aloud.id = base + "-" + n;
      if (lab) lab.htmlFor = aloud.id;
    }
    try { if (localStorage.getItem("sw-talk-aloud") === "0") aloud.checked = false; } catch (e) {}
    aloud.addEventListener("change", function () {
      try { localStorage.setItem("sw-talk-aloud", aloud.checked ? "1" : "0"); } catch (e) {}
      if (!aloud.checked && now() === "speaking") { speechSynthesis.cancel(); wait(); }
    });

    function now() { return el.getAttribute("data-state"); }
    // One button for the person's part of the turn, in the same place
    // throughout, so pressing it never loses the focus.
    function state(s, words) {
      el.setAttribute("data-state", s);
      if (words !== undefined && said.textContent !== words) said.textContent = words;
      if (TURN[s]) {
        turn.textContent = TURN[s];
        turn.hidden = false;
        if (s === "writing" || s === "answering") turn.setAttribute("aria-disabled", "true"); else turn.removeAttribute("aria-disabled");
      } else {
        if (document.activeElement === turn) toggle.focus();
        turn.hidden = true;
      }
    }
    function level() {
      var buf = new Float32Array(analyser.fftSize);
      analyser.getFloatTimeDomainData(buf);
      var sum = 0;
      for (var i = 0; i < buf.length; i++) sum += buf[i] * buf[i];
      return Math.sqrt(sum / buf.length);
    }
    // The microphone is held only while it is about to be used: when voice
    // mode waits, pauses or ends it is let go, and the browser's own sign
    // that it is on goes out.
    function release() {
      if (stream) stream.getTracks().forEach(function (t) { t.stop(); });
      if (ctx) ctx.close();
      stream = ctx = analyser = null;
    }
    function open(then) {
      if (stream) { then(); return; }
      state(now(), "Asking for the microphone.");
      navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: true, noiseSuppression: true } }).then(function (s) {
        if (!active) { s.getTracks().forEach(function (t) { t.stop(); }); return; }
        stream = s;
        var Ctx = window.AudioContext || window.webkitAudioContext;
        ctx = new Ctx();
        analyser = ctx.createAnalyser();
        analyser.fftSize = 1024;
        ctx.createMediaStreamSource(s).connect(analyser);
        if (!voiceWaited && window.speechSynthesis) { voiceWaited = true; speechSynthesis.getVoices(); }
        then();
      }, function (e) {
        active = false;
        toggle.textContent = "Start voice mode";
        state("off", e && e.name === "NotAllowedError"
          ? "The microphone was not allowed, so voice mode cannot listen. It can be allowed in the browser's settings for this site."
          : "The microphone could not be used, so voice mode cannot listen.");
      });
    }
    function listen() {
      if (!active) return;
      if (!stream) { open(listen); return; }
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
        var t = Date.now(), v = level();
        if (t - began < 400) { floor += v; samples++; return; }
        var base = Math.max(0.004, floor / Math.max(1, samples));
        if (v > base * 3 && v > 0.012) {
          if (!speaking) { speaking = true; spokeAt = t; state("hearing"); }
          quietSince = 0;
        } else if (speaking) {
          if (!quietSince) quietSince = t;
          if (t - quietSince > QUIET_END && t - spokeAt > 500) finish();
        }
        if (!speaking && t - began > NOTHING) pause("Paused: nothing was heard for 30 seconds.");
        if (t - began > LONGEST) finish();
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
        return sw.speech.toWav(blob).then(function (wav) {
          return fetch(el.getAttribute("data-action"), { method: "POST", body: wav, headers: { "Content-Type": "audio/wav" }, credentials: "same-origin" });
        }).then(function (r) {
          return r.json().then(function (b) { if (!r.ok) throw new Error(b.error || "It could not be written down."); return (b.text || "").trim(); });
        }).then(function (text) {
          if (!active) return;
          if (!text) { state("listening", "No words were heard. Listening again."); setTimeout(listen, 600); return; }
          ta.value = text;
          ta.dispatchEvent(new Event("input", { bubbles: true }));
          // What was heard is said back, so a wrong word is caught; the
          // chat's own status says the assistant is working.
          state("answering", "Sent: “" + (text.length > 150 ? text.slice(0, 149) + "…" : text) + "”");
          if (form.requestSubmit) form.requestSubmit(); else form.submit();
        });
      }).catch(function (e) { pause((e && e.message) || "Something went wrong, so voice mode paused."); });
    }
    function wait() {
      release();
      state("waiting", "Your turn. Press Speak when you are ready.");
    }
    function speak(text) {
      var voice = localVoice();
      novoice.hidden = !!voice || !aloud.checked;
      if (!aloud.checked || !voice || !text) { wait(); return; }
      state("speaking", "Reading the reply.");
      var list = pieces(text), i = 0, quietFor = 0;
      // Some browsers now and then never say a reading has ended: when
      // nothing has been spoken for a while, it has.
      clearInterval(timer);
      timer = setInterval(function () {
        if (now() !== "speaking") { clearInterval(timer); return; }
        quietFor = speechSynthesis.speaking || speechSynthesis.pending ? 0 : quietFor + 500;
        if (quietFor >= 2000) { clearInterval(timer); speechSynthesis.cancel(); listen(); }
      }, 500);
      function next() {
        if (!active || now() !== "speaking") return;
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
    function pause(why) {
      stopRecorder();
      if (window.speechSynthesis) speechSynthesis.cancel();
      release();
      active = false;
      toggle.textContent = "Continue voice mode";
      state("paused", why + " The microphone is off. Press Continue voice mode to talk again.");
    }
    function end() {
      active = false;
      stopRecorder();
      if (window.speechSynthesis) speechSynthesis.cancel();
      release();
      toggle.textContent = "Start voice mode";
      state("off", "Voice mode is off. The microphone is off.");
    }
    function start() {
      active = true;
      toggle.textContent = "End voice mode";
      listen();
    }
    toggle.addEventListener("click", function () {
      if (active) end(); else start();
    });
    turn.addEventListener("click", function () {
      var s = now();
      if (s === "listening" || s === "hearing") finish();
      else if (s === "speaking") { speechSynthesis.cancel(); listen(); }
      else if (s === "waiting") listen();
    });
    document.addEventListener("keydown", function (e) { if (e.key === "Escape" && (active || stream)) end(); });
    sw.on("turn-done", function (d) {
      if (!active || now() !== "answering") return;
      d = d || {};
      var msg = d.id && document.getElementById("msg-" + d.id);
      var body = msg && msg.querySelector(".sw-message__body");
      speak(body ? body.innerText.trim() : "");
    });
  }
  sw.arm("[data-component=talk]", arm);
})();
