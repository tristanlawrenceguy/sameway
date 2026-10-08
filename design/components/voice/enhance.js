// The microphone, where this browser can use it.
//
// Hidden without this script, and where the page is not secure or the
// browser cannot record: nothing offers what cannot be done. The
// microphone is asked for only when the button is pressed, and one press
// starts it and the next stops it (no holding, no swiping).
//
// record: the recording goes into the form's file field, named for when it
// was made, and waits there for the form's own button.
// dictate: the recording is turned into plain sound here, written down on
// the computer that hosts the workspace, and the words are put into the
// text field where the cursor was, to be read and changed before sending.
// Dictation stops by itself after five minutes and keeps what was said;
// pressing Dictate again carries on.
(function () {
  "use strict";
  var LONGEST = 5 * 60;
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
  // Why the microphone could not start, in words, with what still works.
  function refused(e, mode) {
    var still = mode === "dictate" ? " The message can still be typed." : " A file can still be chosen.";
    var name = e && e.name;
    if (name === "NoSoundShared") {
      return "What was shared had no sound, so nothing was recorded. Press Record again and tick Share audio; for an app such as Teams or Zoom, share the entire screen." + still;
    }
    if (name === "ShareRefused") {
      return "Nothing was shared, so nothing was recorded. Press Record and share the call's tab, or the entire screen, with Share audio ticked; or untick the computer's sound." + still;
    }
    if (name === "NotAllowedError" || name === "SecurityError") {
      return "The microphone was not allowed, so nothing was recorded. It can be allowed in the browser's settings for this site." + still;
    }
    if (name === "NotFoundError" || name === "OverconstrainedError") {
      return "No microphone was found, so nothing was recorded. Connect one and press again." + still;
    }
    if (name === "NotReadableError" || name === "AbortError") {
      return "The microphone could not be started. Another program may be using it; close that and press again." + still;
    }
    return "The microphone could not be used, so nothing was recorded." + still;
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
    // The computer's sound, offered where the browser can give it.
    var also = el.querySelector(".sw-voice__also");
    if (also && window.swSpeech.canHearComputer && window.swSpeech.canHearComputer()) also.hidden = false;
    var failed = "It could not be written down, so nothing was put in the message. Press Dictate to try again, or type it.";
    function state(s) {
      el.setAttribute("data-state", s);
      if (s === "writing") btn.setAttribute("aria-disabled", "true"); else btn.removeAttribute("aria-disabled");
    }
    function fill(blob, seconds, why, voices) {
      var file = new File([blob], "Recording " + stamp(new Date()) + "." + window.swSpeech.ext(blob.type), { type: blob.type });
      // Who was heard each second goes with it, for naming the transcript.
      if (target.form) {
        var v = target.form.querySelector("input[name=voices]");
        if (!v) { v = document.createElement("input"); v.type = "hidden"; v.name = "voices"; target.form.appendChild(v); }
        v.value = voices || "";
      }
      var dt = new DataTransfer();
      dt.items.add(file);
      target.files = dt.files;
      target.dispatchEvent(new Event("change", { bubbles: true }));
      var send = target.form && target.form.querySelector("[type=submit]");
      var name = send ? send.textContent.trim() : "the form's button";
      said.textContent = why + "A recording of " + words(seconds) + " is ready. Press " + name + " to add it.";
    }
    function dictate(blob, why) {
      if (!blob.size) { said.textContent = why + "No words were heard, so nothing was put in the message."; return; }
      state("writing");
      said.textContent = why + "Writing down what you said.";
      window.swSpeech.toWav(blob).then(function (wav) {
        return fetch(el.getAttribute("data-action"), { method: "POST", body: wav, headers: { "Content-Type": "audio/wav" }, credentials: "same-origin" });
      }).then(function (r) {
        return r.json().catch(function () { return {}; }).then(function (body) {
          if (!r.ok) { var e = new Error(body.error ? body.error + " Nothing was put in the message." : failed); e.plain = true; throw e; }
          return body.text || "";
        });
      }).then(function (text) {
        state("idle");
        if (!text) { said.textContent = "No words were heard, so nothing was put in the message."; return; }
        var at = typeof target.selectionStart === "number" ? target.selectionStart : target.value.length;
        var before = target.value.slice(0, at), after = target.value.slice(at);
        var put = (before && !/\s$/.test(before) ? " " : "") + text + (after && !/^\s/.test(after) ? " " : "");
        target.value = before + put + after;
        // The cursor goes after the new words, so the next dictation follows them.
        if (target.setSelectionRange) try { target.setSelectionRange(at + put.length, at + put.length); } catch (e) {}
        target.dispatchEvent(new Event("input", { bubbles: true }));
        var label = target.labels && target.labels[0] ? target.labels[0].textContent.replace("(required)", "").trim() : "the message";
        said.textContent = why + "Written into " + label + ". Read it, then send it.";
      }).catch(function (e) {
        state("idle");
        // A program's own error text (a network or parse failure) is not passed on.
        said.textContent = why + (e && e.plain ? e.message : failed);
      });
    }
    // finish stops the microphone; why says so when it was not the person.
    function finish(why) {
      if (!rec) return;
      var r = rec, seconds = Math.min((Date.now() - started) / 1000, LONGEST);
      rec = null;
      clearInterval(tick);
      time.hidden = true;
      word.textContent = idle;
      state("idle");
      var voices = r.voices ? r.voices() : "";
      r.stop().then(function (blob) {
        if (mode === "dictate") dictate(blob, why); else fill(blob, seconds, why, voices);
      });
    }
    btn.addEventListener("click", function () {
      if (el.getAttribute("data-state") === "writing") return;
      if (rec) { finish(""); return; }
      said.textContent = "";
      var computer = !!(also && !also.hidden && also.querySelector("input").checked);
      window.swSpeech.record({ computer: computer }).then(function (r) {
        rec = r;
        r.onend = function () { finish(computer ? "The microphone or the shared sound stopped, so the recording ended. " : "The microphone stopped, so the recording ended. "); };
        started = Date.now();
        word.textContent = mode === "dictate" ? "Stop dictating" : "Stop recording";
        state("recording");
        var of = mode === "dictate" ? " of " + clock(LONGEST) : "";
        time.textContent = clock(0) + of;
        time.hidden = false;
        tick = setInterval(function () {
          var s = (Date.now() - started) / 1000;
          time.textContent = clock(Math.min(s, LONGEST)) + of;
          if (mode === "dictate" && s >= LONGEST) finish("Dictation stops after 5 minutes; press Dictate to go on. ");
        }, 500);
        // Short, so little of it is caught by the microphone it describes.
        said.textContent = mode === "dictate" ? "Listening, for up to 5 minutes." : "Recording.";
      }, function (e) {
        said.textContent = refused(e, mode);
      });
    });
  }
  sw.arm("[data-component=voice]", arm);
})();
