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
  // record starts the microphone and resolves to something that stops it:
  // stop() resolves to the recording, as a Blob of the type recorded.
  function record() {
    return navigator.mediaDevices.getUserMedia({ audio: true }).then(function (stream) {
      var type = bestType(), chunks = [];
      var rec = type ? new MediaRecorder(stream, { mimeType: type }) : new MediaRecorder(stream);
      rec.addEventListener("dataavailable", function (e) { if (e.data && e.data.size) chunks.push(e.data); });
      rec.start(1000);
      return {
        type: rec.mimeType || type || "audio/webm",
        stop: function () {
          return new Promise(function (ok) {
            rec.addEventListener("stop", function () {
              stream.getTracks().forEach(function (t) { t.stop(); });
              ok(new Blob(chunks, { type: rec.mimeType || type || "audio/webm" }));
            }, { once: true });
            rec.stop();
          });
        }
      };
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
  window.swSpeech = { supported: supported, record: record, toWav: toWav, ext: ext };
})();
