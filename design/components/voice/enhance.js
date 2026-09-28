// The microphone, where this browser can use it.
//
// Hidden without this script, and where the page is not secure or the
// browser cannot record: nothing offers what cannot be done. The
// microphone is asked for only when the button is pressed.
//
// record: the recording goes into the form's file field, named for when it
// was made, and waits there for the form's own button.
// dictate: the recording is turned into plain sound here, written down on
// the computer that hosts the workspace, and the words are put into the
// text field where the cursor was, to be read and changed before sending.
(function () {
  "use strict";
  function clock(s) { return Math.floor(s / 60) + ":" + String(Math.floor(s % 60)).padStart(2, "0"); }
  function words(s) {
    s = Math.max(1, Math.round(s));
    var m = Math.floor(s / 60), x = s % 60, out = [];
    if (m) out.push(m + (m === 1 ? " minute" : " minutes"));
    if (x) out.push(x + (x === 1 ? " second" : " seconds"));
    return out.join(" ");
  }
  function stamp(d) {
    var p = function (n) { return String(n).padStart(2, "0"); };
    return d.getFullYear() + "-" + p(d.getMonth() + 1) + "-" + p(d.getDate()) + " " + p(d.getHours()) + "." + p(d.getMinutes());
  }
  function arm(el) {
    if (el._armed || !window.swSpeech || !window.swSpeech.supported()) return;
    var target = document.getElementById(el.getAttribute("data-target"));
    if (!target) return;
    el._armed = true;
    el.hidden = false;
    var mode = el.getAttribute("data-mode");
    var btn = el.querySelector(".sw-voice__toggle"), word = el.querySelector(".sw-voice__word");
    var time = el.querySelector(".sw-voice__time"), said = el.querySelector(".sw-voice__said");
    var idle = word.textContent, rec = null, started = 0, tick = null;
    function state(s) { el.setAttribute("data-state", s); }
    function stopped() {
      clearInterval(tick);
      time.hidden = true;
      word.textContent = idle;
      state("idle");
    }
    function fill(blob, seconds) {
      var file = new File([blob], "Recording " + stamp(new Date()) + "." + window.swSpeech.ext(blob.type), { type: blob.type });
      var dt = new DataTransfer();
      dt.items.add(file);
      target.files = dt.files;
      target.dispatchEvent(new Event("change", { bubbles: true }));
      var send = target.form && target.form.querySelector("[type=submit]");
      var name = send ? send.textContent.trim() : "the form's button";
      said.textContent = "A recording of " + words(seconds) + " is ready. Press " + name + " to add it.";
    }
    function dictate(blob) {
      state("writing");
      said.textContent = "Writing down what you said.";
      window.swSpeech.toWav(blob).then(function (wav) {
        return fetch(el.getAttribute("data-action"), { method: "POST", body: wav, headers: { "Content-Type": "audio/wav" }, credentials: "same-origin" });
      }).then(function (r) {
        return r.json().then(function (body) { if (!r.ok) throw new Error(body.error || "It could not be written down."); return body.text || ""; });
      }).then(function (text) {
        state("idle");
        if (!text) { said.textContent = "No words were heard."; return; }
        var at = typeof target.selectionStart === "number" ? target.selectionStart : target.value.length;
        var before = target.value.slice(0, at), after = target.value.slice(at);
        var gap = before && !/\s$/.test(before) ? " " : "";
        target.value = before + gap + text + (after && !/^\s/.test(after) ? " " : "") + after;
        target.dispatchEvent(new Event("input", { bubbles: true }));
        var label = target.labels && target.labels[0] ? target.labels[0].textContent.replace("(required)", "").trim() : "the message";
        said.textContent = "Written into " + label + ". Read it, then send it.";
      }, function (e) {
        state("idle");
        said.textContent = e.message || "It could not be written down.";
      });
    }
    btn.addEventListener("click", function () {
      if (el.getAttribute("data-state") === "writing") return;
      if (!rec) {
        said.textContent = "";
        window.swSpeech.record().then(function (r) {
          rec = r;
          started = Date.now();
          word.textContent = mode === "dictate" ? "Stop dictating" : "Stop recording";
          state("recording");
          time.textContent = "0:00";
          time.hidden = false;
          tick = setInterval(function () { time.textContent = clock((Date.now() - started) / 1000); }, 500);
          said.textContent = mode === "dictate" ? "Listening. Press Stop dictating when you have finished." : "Recording. Press Stop recording to finish.";
        }, function (e) {
          said.textContent = e && e.name === "NotAllowedError"
            ? "The microphone was not allowed, so nothing was recorded. It can be allowed in the browser's settings for this site."
            : "The microphone could not be used, so nothing was recorded.";
        });
        return;
      }
      var r = rec, seconds = (Date.now() - started) / 1000;
      rec = null;
      stopped();
      r.stop().then(function (blob) {
        if (mode === "dictate") dictate(blob); else fill(blob, seconds);
      });
    });
  }
  function init() { document.querySelectorAll("[data-component=voice]").forEach(arm); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
  document.addEventListener("sw:refresh", init);
})();
