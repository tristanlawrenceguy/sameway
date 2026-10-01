// Sound from this device: recording from its microphone, and any sound
// its browser can play turned into the plain sound speech-to-text reads.
//
// window.swSpeech is shared by the voice component (record and dictate)
// and the audio component (writing a recording down). Recording needs a
// secure page (https, or this computer) and a browser that records; where
// either is missing, supported() says so and nothing offers to record.
(function () {
  "use strict";
  function bestType() {
    if (!window.MediaRecorder || !MediaRecorder.isTypeSupported) return "";
    var types = ["audio/webm;codecs=opus", "audio/ogg;codecs=opus", "audio/mp4", "audio/webm"];
    for (var i = 0; i < types.length; i++) if (MediaRecorder.isTypeSupported(types[i])) return types[i];
    return "";
  }
  function ext(type) {
    if (/mp4/.test(type)) return "m4a";
    if (/ogg/.test(type)) return "ogg";
    return "webm";
  }
  function supported() {
    return !!(window.isSecureContext && navigator.mediaDevices && navigator.mediaDevices.getUserMedia && window.MediaRecorder);
  }
  // canHearComputer says whether this browser can be given the computer's
  // sound, or a tab's: what the other people on a call say.
  function canHearComputer() {
    return supported() && !!navigator.mediaDevices.getDisplayMedia && !!(window.AudioContext || window.webkitAudioContext);
  }
  // computerSound asks the person to share a tab or the screen with its
  // sound, keeps the sound and lets the picture go. The browser cannot be
  // asked for sound alone; a share without sound is refused.
  function computerSound() {
    return navigator.mediaDevices.getDisplayMedia({
      video: true, audio: { echoCancellation: false, noiseSuppression: false },
      systemAudio: "include", selfBrowserSurface: "exclude", restrictOwnAudio: true
    }).then(function (shared) {
      shared.getVideoTracks().forEach(function (t) { t.stop(); });
      if (!shared.getAudioTracks().length) {
        var e = new Error("no sound shared");
        e.name = "NoSoundShared";
        throw e;
      }
      return shared;
    }, function (e) {
      var r = new Error("nothing shared");
      r.name = "ShareRefused";
      throw r;
    });
  }
  // mix is the microphone and the computer's sound as one stream, and who
  // was heard each second: t when the call is sounding (them), m when only
  // the microphone is (me), . for neither. The call is read from its own
  // sound, never the microphone's, so a call heard through speakers is
  // still them. The server names the transcript's lines from it.
  function mix(mic, shared) {
    var Ctx = window.AudioContext || window.webkitAudioContext, ctx = new Ctx(), out = ctx.createMediaStreamDestination();
    var me = ctx.createMediaStreamSource(mic), them = ctx.createMediaStreamSource(new MediaStream(shared.getAudioTracks()));
    me.connect(out);
    them.connect(out);
    function meter(src) {
      var a = ctx.createAnalyser();
      a.fftSize = 2048;
      src.connect(a);
      var buf = new Float32Array(a.fftSize);
      return function () {
        a.getFloatTimeDomainData(buf);
        var sum = 0;
        for (var i = 0; i < buf.length; i++) sum += buf[i] * buf[i];
        return Math.sqrt(sum / buf.length);
      };
    }
    var meLevel = meter(me), themLevel = meter(them), seen = "", m = 0, t = 0, n = 0;
    var tick = setInterval(function () {
      var tl = themLevel(), ml = meLevel();
      // Floors measured on the AMI meeting corpus (internal/speech/ami_bench_test.go):
      // a call is near silent between speakers, so a low floor finds them.
      if (tl > 0.002) t++; else if (ml > 0.003) m++;
      if (++n === 4) {
        seen += t > 0 && t >= m ? "t" : m > 0 ? "m" : ".";
        m = t = n = 0;
      }
    }, 250);
    return { stream: out.stream, ctx: ctx, voices: function () { return seen; }, stop: function () { clearInterval(tick); } };
  }
  // record starts the microphone and resolves to something that stops it:
  // stop() resolves to the recording, as a Blob of the type recorded.
  // onend, if set, is called when the microphone stops by itself (taken
  // away, or its permission withdrawn); stop() still gives what was heard.
  // With opts.computer, the computer's sound is recorded with it.
  function record(opts) {
    var shared = null;
    var asked = opts && opts.computer ? computerSound().then(function (s) { shared = s; }) : Promise.resolve();
    return asked.then(function () {
      return navigator.mediaDevices.getUserMedia({ audio: true });
    }).then(function (mic) {
      var mixed = shared ? mix(mic, shared) : null, stream = mixed ? mixed.stream : mic;
      var all = shared ? mic.getTracks().concat(shared.getTracks()) : mic.getTracks();
      var type = bestType(), chunks = [];
      var rec = type ? new MediaRecorder(stream, { mimeType: type }) : new MediaRecorder(stream);
      rec.addEventListener("dataavailable", function (e) { if (e.data && e.data.size) chunks.push(e.data); });
      rec.start(1000);
      var api = {
        type: rec.mimeType || type || "audio/webm",
        onend: null,
        // voices is who was heard each second, when the call was recorded.
        voices: function () { return mixed ? mixed.voices() : ""; },
        stop: function () {
          var done = function () {
            all.forEach(function (t) { t.stop(); });
            if (mixed) { mixed.stop(); mixed.ctx.close(); }
            return new Blob(chunks, { type: rec.mimeType || type || "audio/webm" });
          };
          if (rec.state === "inactive") return Promise.resolve(done());
          return new Promise(function (ok) {
            rec.addEventListener("stop", function () { ok(done()); }, { once: true });
            rec.stop();
          });
        }
      };
      all.forEach(function (t) {
        t.addEventListener("ended", function () { if (api.onend) api.onend(); });
      });
      return api;
    }, function (e) {
      if (shared) shared.getTracks().forEach(function (t) { t.stop(); });
      throw e;
    });
  }
  function decode(ctx, data) {
    return new Promise(function (ok, bad) {
      var p = ctx.decodeAudioData(data, ok, bad);
      if (p && p.then) p.then(ok, bad);
    });
  }
  // toWav turns a recording (a Blob, or the address of one) into 16 kHz,
  // one channel, 16-bit WAV: what the speech engine reads, made here so
  // the computer that writes it down needs no other program.
  function toWav(source) {
    var got = typeof source === "string"
      ? fetch(source, { credentials: "same-origin" }).then(function (r) { return r.arrayBuffer(); })
      : source.arrayBuffer();
    return got.then(function (data) {
      var Ctx = window.OfflineAudioContext || window.webkitOfflineAudioContext, ctx;
      try { ctx = new Ctx(1, 1, 16000); } catch (e) { ctx = new Ctx(1, 1, 44100); }
      return decode(ctx, data);
    }).then(function (buf) {
      var n = buf.length, ch = buf.numberOfChannels, mono = new Float32Array(n);
      for (var c = 0; c < ch; c++) {
        var d = buf.getChannelData(c);
        for (var i = 0; i < n; i++) mono[i] += d[i] / ch;
      }
      if (buf.sampleRate !== 16000) {
        var step = buf.sampleRate / 16000, out = new Float32Array(Math.floor(n / step));
        for (var j = 0; j < out.length; j++) {
          var pos = j * step, k = Math.floor(pos), f = pos - k;
          out[j] = k + 1 < n ? mono[k] * (1 - f) + mono[k + 1] * f : mono[n - 1];
        }
        mono = out;
      }
      var bytes = new DataView(new ArrayBuffer(44 + mono.length * 2));
      function text(at, s) { for (var q = 0; q < s.length; q++) bytes.setUint8(at + q, s.charCodeAt(q)); }
      text(0, "RIFF"); bytes.setUint32(4, 36 + mono.length * 2, true); text(8, "WAVEfmt ");
      bytes.setUint32(16, 16, true); bytes.setUint16(20, 1, true); bytes.setUint16(22, 1, true);
      bytes.setUint32(24, 16000, true); bytes.setUint32(28, 32000, true); bytes.setUint16(32, 2, true);
      bytes.setUint16(34, 16, true); text(36, "data"); bytes.setUint32(40, mono.length * 2, true);
      for (var s = 0; s < mono.length; s++) bytes.setInt16(44 + s * 2, Math.max(-1, Math.min(1, mono[s])) * 32767, true);
      return new Blob([bytes.buffer], { type: "audio/wav" });
    });
  }
  window.swSpeech = { supported: supported, canHearComputer: canHearComputer, record: record, toWav: toWav, ext: ext };
})();
