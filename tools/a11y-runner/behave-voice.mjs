// The microphone (design/components/voice and talk, design/base/28-speech.js),
// with a made-up one: the page is told it is secure and given a microphone
// that hums, or refuses as the case says (window.__refuse). Writing down
// what was said is the server's, answered by the case.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { open, page } from "./harness.mjs";

const example = (component, name) => readFileSync(fileURLToPath(new URL(`../../design/components/${component}/examples/${name}.html`, import.meta.url)), "utf8");

// The microphone is quiet until window.__loud() says the person speaks.
const microphone = () => {
  Object.defineProperty(window, "isSecureContext", { get: () => true });
  if (!navigator.mediaDevices) Object.defineProperty(navigator, "mediaDevices", { value: {} });
  navigator.mediaDevices.getUserMedia = async () => {
    if (window.__refuse) throw new DOMException("refused", window.__refuse);
    const ctx = new AudioContext();
    const hum = ctx.createOscillator(), level = ctx.createGain(), out = ctx.createMediaStreamDestination();
    level.gain.value = 0;
    hum.connect(level).connect(out);
    window.__loud = () => { level.gain.value = 1; };
    hum.start();
    window.__mic = out.stream;
    return out.stream;
  };
};
// stubbed makes the sound into a WAV at once: the browser's own decoding
// of a made-up hum is not what these cases are about.
const stubbed = (tab) => tab.evaluate(() => { sw.speech.toWav = () => Promise.resolve(new Blob(["wav"], { type: "audio/wav" })); });
const micOff = (tab) => tab.evaluate(() => !window.__mic || window.__mic.getTracks().every((t) => t.readyState === "ended"));

const dictation = page(`<main id="main"><form method="post" action="/chat" onsubmit="event.preventDefault()"><label for="message">Message</label><textarea id="message" name="message"></textarea><button type="submit">Send</button></form>${example("voice", "dictate")}</main>`);
const recording = page(`<main id="main"><form method="post" action="/t/file" onsubmit="event.preventDefault()"><label for="upload">File</label><input type="file" id="upload" name="file"><button type="submit">Upload</button></form>${example("voice", "record").replace(/data-target="[^"]*"/, 'data-target="upload"')}</main>`);
const said = (tab, sel = ".sw-voice__said") => tab.textContent(sel);

export const cases = {
  "voice: a microphone that cannot start says why in words, with what still works": async (browser, check) => {
    const why = { NotAllowedError: /was not allowed.*can be allowed in the browser's settings for this site\./, NotFoundError: /No microphone was found/, NotReadableError: /Another program may be using it/ };
    for (const [name, words] of Object.entries(why)) {
      for (const [html, still] of [[dictation, " The message can still be typed."], [recording, " A file can still be chosen."]]) {
        const tab = await open(browser, html, {}, microphone);
        await tab.evaluate((n) => { window.__refuse = n; }, name);
        await tab.click(".sw-voice__toggle");
        await tab.waitForFunction(() => document.querySelector(".sw-voice__said").textContent);
        const s = await said(tab);
        check(words.test(s) && s.endsWith(still), `${name}: ${s}`);
        await tab.close();
      }
    }
  },

  "voice: dictation says its limit, is unavailable while words are written down, and puts them where the cursor was": async (browser, check) => {
    let answer = null;
    const tab = await open(browser, dictation, { "/dictate": (route) => new Promise((r) => { answer = r; }).then((body) => route.fulfill(body)) }, microphone);
    await stubbed(tab);
    await tab.fill("#message", "Buy  please");
    await tab.evaluate(() => document.getElementById("message").setSelectionRange(4, 4));
    await tab.click(".sw-voice__toggle");
    await tab.waitForFunction(() => document.querySelector(".sw-voice").getAttribute("data-state") === "recording");
    check(await said(tab) === "Listening, for up to 5 minutes.", `the limit is said as it starts: ${await said(tab)}`);
    check(/^0:0\d of 5:00$/.test(await tab.textContent(".sw-voice__time")), `and shown beside the time: ${await tab.textContent(".sw-voice__time")}`);
    await tab.waitForTimeout(1200);
    await tab.click(".sw-voice__toggle");
    await tab.waitForFunction(() => document.querySelector(".sw-voice").getAttribute("data-state") === "writing");
    check(await tab.getAttribute(".sw-voice__toggle", "aria-disabled") === "true", "unavailable, not silent, while the words are written down");
    check(await micOff(tab), "the microphone is let go");
    answer({ contentType: "application/json", body: JSON.stringify({ text: "milk" }) });
    await tab.waitForFunction(() => document.querySelector(".sw-voice").getAttribute("data-state") === "idle");
    const box = await tab.evaluate(() => { const t = document.getElementById("message"); return { value: t.value, at: t.selectionStart }; });
    check(box.value === "Buy milk please" && box.at === "Buy milk".length, `the words go where the cursor was, and it goes after them: ${JSON.stringify(box)}`);
    check(/Written into Message\. Read it, then send it\./.test(await said(tab)), `said: ${await said(tab)}`);
    await tab.close();
  },

  "voice: dictation stops itself at five minutes and keeps what was said; a refusal passes on only plain words": async (browser, check) => {
    let reply = { contentType: "application/json", body: JSON.stringify({ text: "milk" }) };
    const tab = await open(browser, dictation, { "/dictate": (route) => (reply.abort ? route.abort() : route.fulfill(reply)) }, microphone);
    await stubbed(tab);
    await tab.click(".sw-voice__toggle");
    await tab.waitForFunction(() => document.querySelector(".sw-voice").getAttribute("data-state") === "recording");
    await tab.waitForTimeout(1200);
    await tab.evaluate(() => { window.realNow = Date.now; Date.now = () => window.realNow() + 301000; });
    await tab.waitForFunction(() => document.getElementById("message").value === "milk", null, { timeout: 5000 }).catch(() => {});
    check(await tab.inputValue("#message") === "milk", "what was said is kept");
    check(/^Dictation stops after 5 minutes; press Dictate to go on\. /.test(await said(tab)), `and it says how to go on: ${await said(tab)}`);
    for (const [answer, want] of [
      [{ status: 422, contentType: "application/json", body: JSON.stringify({ error: "The recording was too quiet." }) }, "The recording was too quiet. Nothing was put in the message."],
      [{ status: 502, contentType: "text/html", body: "<h1>Bad Gateway</h1> nginx" }, "It could not be written down, so nothing was put in the message. Press Dictate to try again, or type it."],
      [{ abort: true, status: "no answer" }, "It could not be written down, so nothing was put in the message. Press Dictate to try again, or type it."],
    ]) {
      reply = answer;
      await tab.click(".sw-voice__toggle");
      await tab.waitForFunction(() => document.querySelector(".sw-voice").getAttribute("data-state") === "recording");
      await tab.waitForTimeout(1200);
      await tab.click(".sw-voice__toggle");
      await tab.waitForFunction(() => !/Writing|Listening/.test(document.querySelector(".sw-voice__said").textContent));
      check(await said(tab) === want, `${answer.status}: ${await said(tab)}`);
    }
    await tab.close();
  },

  "voice: a microphone taken away ends the recording, which is kept": async (browser, check) => {
    const tab = await open(browser, recording, {}, microphone);
    await tab.click(".sw-voice__toggle");
    await tab.waitForFunction(() => document.querySelector(".sw-voice").getAttribute("data-state") === "recording");
    check(await said(tab) === "Recording.", `the start is said briefly: ${await said(tab)}`);
    await tab.waitForTimeout(1200);
    await tab.evaluate(() => window.__mic.getTracks().forEach((t) => t.dispatchEvent(new Event("ended"))));
    await tab.waitForFunction(() => /ready/.test(document.querySelector(".sw-voice__said").textContent));
    check(/^The microphone stopped, so the recording ended\. A recording of .* is ready\. Press Upload to add it\.$/.test(await said(tab)), `said: ${await said(tab)}`);
    check(await tab.evaluate(() => document.getElementById("upload").files.length) === 1, "the recording is in the form, waiting");
    await tab.close();
  },
};

// Voice mode (talk): one turn button in place, the microphone let go
// whenever it is not about to be used, and what was heard said back.
const chatWithTalk = (talks) => page(`<main id="main"><form method="post" action="/chat" id="chat" onsubmit="event.preventDefault(); window.sent = (window.sent || 0) + 1"><label for="message">Message</label><textarea id="message" name="message"></textarea><button type="submit">Send</button></form>${talks}</main>`);
const talk = example("talk", "default");

Object.assign(cases, {
  "talk: the turn button stays in place, focus goes to Start when it goes, and Hearing you is never said": async (browser, check) => {
    const tab = await open(browser, chatWithTalk(talk), {}, microphone);
    await tab.evaluate(() => {
      window.statuses = [];
      new MutationObserver(() => window.statuses.push(document.querySelector(".sw-talk__state").textContent)).observe(document.querySelector(".sw-talk__state"), { childList: true, characterData: true, subtree: true });
    });
    await tab.click("text=Start voice mode");
    await tab.waitForFunction(() => document.querySelector(".sw-talk").getAttribute("data-state") === "listening");
    check(await tab.evaluate(() => document.querySelectorAll(".sw-talk button:not([hidden])").length) === 2, "Start and one turn button");
    await tab.waitForTimeout(600);
    await tab.evaluate(() => window.__loud());
    await tab.waitForFunction(() => document.querySelector(".sw-talk").getAttribute("data-state") === "hearing");
    check(await tab.isVisible(".sw-talk__heard"), "Hearing you is shown while the person speaks");
    await tab.focus(".sw-talk__turn");
    await tab.keyboard.press("Escape");
    await tab.waitForFunction(() => document.querySelector(".sw-talk").getAttribute("data-state") === "off");
    check(await tab.evaluate(() => document.activeElement.classList.contains("sw-talk__toggle")), "the turn button went, and focus went to the start button");
    check(await micOff(tab), "ended, the microphone is off");
    check(await tab.evaluate(() => window.statuses.every((s) => !/Hearing you/.test(s))), `Hearing you is shown, never put in the status: ${await tab.evaluate(() => window.statuses.join(" | "))}`);
    check(await tab.isVisible("text=Headphones help"), "the hint about headphones is shown");
    await tab.close();
  },

  "talk: a second voice mode gives its checkbox its own id, so its label still presses it": async (browser, check) => {
    const tab = await open(browser, chatWithTalk(talk + talk.replace('id="talk"', 'id="talk2"')), {}, microphone);
    const second = tab.locator(".sw-talk").nth(1);
    await second.locator("label").click();
    const ticks = await tab.evaluate(() => [...document.querySelectorAll(".sw-talk input[type=checkbox]")].map((b) => b.id + ":" + b.checked));
    check(ticks[0].endsWith(":true") && ticks[1].endsWith(":false") && ticks[0].split(":")[0] !== ticks[1].split(":")[0], `the second label presses the second box: ${ticks}`);
    await tab.close();
  },

  "talk: what was heard is said back as it is sent; a reply not read aloud lets the microphone go and waits for Speak": async (browser, check) => {
    const tab = await open(browser, chatWithTalk(talk + `<ol><li id="msg-r1"><div class="sw-message__body"><p>Done: three things planned.</p></div></li></ol>`), {
      "/dictate": (route) => route.fulfill({ contentType: "application/json", body: JSON.stringify({ text: "Plan my week" }) }),
    }, microphone);
    await stubbed(tab);
    await tab.uncheck("text=Read replies aloud");
    await tab.click("text=Start voice mode");
    await tab.waitForFunction(() => document.querySelector(".sw-talk").getAttribute("data-state") === "listening");
    await tab.waitForTimeout(1200);
    await tab.click(".sw-talk__turn");
    await tab.waitForFunction(() => document.querySelector(".sw-talk").getAttribute("data-state") === "answering");
    check(await tab.textContent(".sw-talk__state") === "Sent: “Plan my week”", `said back as it is sent: ${await tab.textContent(".sw-talk__state")}`);
    check(await tab.evaluate(() => window.sent) === 1 && await tab.inputValue("#message") === "Plan my week", "sent through the chat's own form");
    await tab.evaluate(() => sw.emit("turn-done", { id: "r1" }));
    await tab.waitForFunction(() => document.querySelector(".sw-talk").getAttribute("data-state") === "waiting");
    check(await micOff(tab), "a reply not read aloud lets the microphone go");
    check(await tab.textContent(".sw-talk__turn") === "Speak", `and waits for Speak: ${await tab.textContent(".sw-talk__turn")}`);
    await tab.close();
  },

  "talk: pausing lets the microphone go and says so": async (browser, check) => {
    const tab = await open(browser, chatWithTalk(talk), {
      "/dictate": (route) => route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ error: "Writing down is not set up on this computer." }) }),
    }, microphone);
    await stubbed(tab);
    await tab.click("text=Start voice mode");
    await tab.waitForFunction(() => document.querySelector(".sw-talk").getAttribute("data-state") === "listening");
    await tab.waitForTimeout(1200);
    await tab.click(".sw-talk__turn");
    await tab.waitForFunction(() => document.querySelector(".sw-talk").getAttribute("data-state") === "paused");
    check(/The microphone is off\. Press Continue voice mode to talk again\.$/.test(await tab.textContent(".sw-talk__state")), `said: ${await tab.textContent(".sw-talk__state")}`);
    check(await micOff(tab), "and it is");
    await tab.close();
  },
});
