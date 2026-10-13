// Component scripts moved out of design/base into the component's own
// enhance.js (AGENTS.md): what each promises, unchanged by the move.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { open, page, refreshed } from "./harness.mjs";

const alert = `<div class="sw-alert sw-alert--success sw-alert--dismissible" data-component="alert" data-kind="success"><p class="sw-alert__message">Changes made.</p><button type="button" class="sw-alert__close" data-dismiss aria-label="Close message"><span aria-hidden="true">×</span></button></div>`;

const example = (component, name) => readFileSync(fileURLToPath(new URL(`../../design/components/${component}/examples/${name}.html`, import.meta.url)), "utf8");
const sse = (body) => ({ status: 200, headers: { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" }, body });

// wav is a second of silence per length, a recording the browser can play.
function wav(seconds) {
  const rate = 8000, n = rate * seconds, b = Buffer.alloc(44 + n);
  b.write("RIFF", 0); b.writeUInt32LE(36 + n, 4); b.write("WAVEfmt ", 8); b.writeUInt32LE(16, 16); b.writeUInt16LE(1, 20); b.writeUInt16LE(1, 22);
  b.writeUInt32LE(rate, 24); b.writeUInt32LE(rate, 28); b.writeUInt16LE(1, 32); b.writeUInt16LE(8, 34); b.write("data", 36); b.writeUInt32LE(n, 40);
  b.fill(128, 44);
  return b;
}
const player = (id, src, type) => example("media", "default").replace(/id="standup"/, `id="${id}"`).replace(/standup-title/g, `${id}-title`).replace(/<source src="\/files\/r1" type="audio\/mp4">/, `<source src="${src}" type="${type}">`).replace("Monday stand-up</figcaption>", `${id}</figcaption>`);
const here = (who) => `<p class="sw-present" data-component="presence">${who ? `Here with you: <span class="sw-present__who" data-here><span class="sw-person" data-component="person" data-person="3"><span class="sw-person__dot" aria-hidden="true"></span><bdi>${who}</bdi></span>, on this page</span>` : ""}</p>`;

export const cases = {
  "clock: a reminder shows in every clock and is said once, as its name, by the first": async (browser, check) => {
    const ring = `event: ring\ndata: ${JSON.stringify({ id: "r9", title: "Water the plants", text: "It is time." })}\n\n`;
    const tab = await open(browser, page(`<main id="main">${example("clock", "default")}${example("clock", "glance")}</main>`), { "/events": (route) => route.fulfill(sse(ring)) });
    await tab.waitForFunction(() => document.querySelectorAll('.sw-clock__ring[data-id="r9"]').length === 2, null, { timeout: 8000 }).catch(() => {});
    check(await tab.locator('.sw-clock__ring[data-id="r9"]').count() === 2, "it shows in both clocks");
    const said = await tab.evaluate(() => [...document.querySelectorAll("[role=alert], [aria-live]")].map((r) => r.className + ":" + r.textContent.trim()).filter((s) => !s.endsWith(":")));
    check(said.length === 1 && said[0].endsWith(":Reminder: Water the plants"), `said once, as its name: ${JSON.stringify(said)}`);
    check(await tab.evaluate(() => !document.querySelector(".sw-clock__ringing[role], .sw-clock__ringing[aria-live]")), "the list of what rings is not read out whole");
    await tab.close();
  },

  "presence: someone coming onto this page is said once, politely; leaving is never said": async (browser, check) => {
    const steps = ["", "Hana", "", "Hana"];
    const tab = await open(browser, (n) => page(`<div class="sw-shell"><main id="main">${here(steps[Math.min(n, steps.length) - 1])}</main></div>`));
    // Everything the status says, as it is said.
    await tab.evaluate(() => { window.said = []; const r = document.querySelector("[data-presence-said]"); new MutationObserver(() => r.textContent && window.said.push(r.textContent)).observe(r, { childList: true, characterData: true, subtree: true }); });
    for (let i = 1; i < steps.length; i++) {
      const done = refreshed(tab);
      await tab.evaluate(() => sw.refresh(0));
      check(await done, `the page followed itself, ${i}`);
    }
    await tab.waitForTimeout(100);
    const said = await tab.evaluate(() => window.said);
    check(said.length === 1 && said[0] === "Hana is on this page too.", `her arrival is said, once, and her leaving and coming back are not: ${JSON.stringify(said)}`);
    check(await tab.evaluate(() => { const r = document.querySelector("[data-presence-said]"); return r.getAttribute("role") === "status" && r.getAttribute("aria-live") !== "assertive"; }), "through a polite status");
    await tab.close();
  },

  "media: one recording playing pauses another, its seek bar holds still under focus, and one it cannot play says so": async (browser, check) => {
    const sound = (route) => route.fulfill({ contentType: "audio/wav", body: wav(20) });
    const tab = await open(browser, page(`<main id="main">${player("one", "/files/one.wav", "audio/wav")}${player("two", "/files/two.wav", "audio/wav")}${player("odd", "/files/odd", "audio/mp4")}</main>`), {
      "/files/one.wav": sound, "/files/two.wav": sound, "/files/odd": (route) => route.fulfill({ status: 404, body: "" }),
    });
    const playing = () => tab.evaluate(() => [...document.querySelectorAll("audio")].map((a) => !a.paused));
    await tab.locator("#one").getByRole("button", { name: /^Play/ }).click();
    await tab.waitForFunction(() => !document.querySelector("#one audio").paused);
    const seek = tab.locator("#one .sw-media__seek");
    await seek.focus();
    const held = await seek.inputValue();
    await tab.waitForTimeout(2500);
    const t = await tab.evaluate(() => document.querySelector("#one audio").currentTime);
    check(t > 1 && await seek.inputValue() === held, `the seek bar holds still under focus while it plays on: ${held} then ${await seek.inputValue()}, at ${t}s`);
    await tab.locator("#two").getByRole("button", { name: /^Play/ }).click();
    await tab.waitForFunction(() => !document.querySelector("#two audio").paused);
    await tab.waitForFunction(() => document.querySelector("#one audio").paused, null, { timeout: 2000 }).catch(() => {});
    check((await playing()).slice(0, 2).join() === "false,true", `the second paused the first: ${await playing()}`);
    await tab.waitForFunction(() => /cannot play/.test(document.getElementById("odd").textContent), null, { timeout: 5000 }).catch(() => {});
    check(/This browser cannot play odd\. Download it to play it elsewhere\./.test(await tab.textContent("#odd")), `a recording it cannot play says so: ${await tab.textContent("#odd .sw-media__trouble").catch(() => "")}`);
    await tab.close();
  },

  "alert: the outcome takes focus on arrival, and closing it gives focus back to the page": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main" tabindex="-1"><h1>Page</h1><div class="sw-outcome" id="outcome" tabindex="-1" data-outcome="done">${alert}</div></main>`));
    await tab.waitForFunction(() => document.activeElement && document.activeElement.id === "outcome");
    await tab.keyboard.press("Tab");
    check(await tab.evaluate(() => document.activeElement.getAttribute("aria-label")) === "Close message", "Tab reaches Close");
    await tab.keyboard.press("Enter");
    check(await tab.locator("#outcome").count() === 0, "the outcome is gone");
    check(await tab.evaluate(() => document.activeElement.id) === "main", "focus is back in the page, not nowhere");
    await tab.close();
  },

  "alert: one an outcome brings later closes too": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main" tabindex="-1"><h1>Page</h1></main>`));
    await tab.evaluate((html) => { document.getElementById("main").insertAdjacentHTML("beforeend", html); sw.emit("refresh"); }, alert);
    await tab.click("[data-dismiss]");
    check(await tab.locator(".sw-alert").count() === 0, "closed");
    await tab.close();
  },
};
