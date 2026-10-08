// Scripts that arm what the page brings in (sw.arm in 00-sw.js): a block a
// refresh brings gets its Edit once, a question the assistant asks at the
// end of a turn is answered without leaving the page, and an upload form
// says what is wrong with a file before it is sent.
import { readFileSync } from "node:fs";
import { open, page, refreshed } from "./harness.mjs";

const block = (id) => `<li class="sw-block" data-block-id="${id}" data-block-component="text"><div class="sw-bar"></div><p data-prop="text">Words of ${id}</p></li>`;
const shell = (blocks) => `<div class="sw-shell"><main id="main" tabindex="-1"><ol class="sw-canvas">${blocks}</ol></main></div>`;

const chat = (proposal) => `<div class="sw-shell"><main id="main" tabindex="-1">${proposal}
<p class="sw-status sw-status--idle" data-component="status" data-state="idle" role="status" aria-live="polite" id="chat-status"><span class="sw-status__text"></span></p>
<form method="post" action="/chat" class="sw-compose" data-busy-target="chat-status"><label for="m">Message</label><textarea id="m" name="message"></textarea><button type="submit">Send</button></form>
</main></div>`;
const proposal = (id) => `<div class="sw-proposals"><div class="sw-proposal" data-component="proposal" role="group" aria-labelledby="${id}-ask" id="${id}"><p class="sw-proposal__summary" id="${id}-ask">Remove the link?</p><div class="sw-proposal__answers"><form method="post" action="/proposal/${id}/accept"><button type="submit" class="sw-button">Yes, remove it</button></form><form method="post" action="/proposal/${id}/dismiss"><button type="submit" class="sw-button">No, leave it</button></form></div></div></div>`;

export const cases = {
  "edit: every block has one Edit, and one a refresh brings gets its own": async (browser, check) => {
    const tab = await open(browser, (n) => page(shell(block("b1") + (n > 1 ? block("b2") : ""))));
    const edits = () => tab.evaluate(() => [...document.querySelectorAll("[data-block-id]")].map((b) => b.getAttribute("data-block-id") + ":" + b.querySelectorAll("[data-edit]").length).join());
    check(await edits() === "b1:1", `on load: ${await edits()}`);
    for (let i = 0; i < 2; i++) {
      const done = refreshed(tab);
      await tab.evaluate(() => sw.refresh(0));
      check(await done, "the page followed itself");
    }
    check(await edits() === "b1:1,b2:1", `after two refreshes: ${await edits()}`);
    check(!tab.errors.length, `no errors: ${tab.errors.join("; ")}`);
    await tab.close();
  },

  "proposal: an answer is sent in place, every question goes, and focus goes to the message box": async (browser, check) => {
    const posted = [];
    const tab = await open(browser, page(chat(proposal("p1"))), {
      "/proposal/p1/accept": (route, url) => { posted.push(url.pathname); return route.fulfill({ contentType: "text/html", body: page("<p>done</p>") }); },
    });
    await tab.evaluate(() => { window.followed = 0; sw.refresh = () => { window.followed++; }; });
    await tab.click("text=Yes, remove it");
    await tab.waitForFunction(() => !document.querySelector(".sw-proposal"));
    check(await tab.evaluate(() => window.followed) === 1, "the page is asked to follow the change");
    check(posted.join() === "/proposal/p1/accept", `sent once, to its form: ${posted.join()}`);
    check(tab.url().endsWith("/"), `the page stayed: ${tab.url()}`);
    check(await tab.textContent("#chat-status .sw-status__text") === "Done: Yes, remove it.", `the status says what it did: ${await tab.textContent("#chat-status")}`);
    check(await tab.evaluate(() => document.activeElement.id) === "m", "focus is in the message box");
    await tab.close();
  },

  "proposal: a refusal says why and gives the buttons back": async (browser, check) => {
    const tab = await open(browser, page(chat(proposal("p1"))), {
      "/proposal/p1/accept": (route) => route.fulfill({ contentType: "text/html", body: page(`<div class="sw-outcome" data-outcome="failed">That question has gone.</div>`) }),
    });
    await tab.click("text=Yes, remove it");
    await tab.waitForFunction(() => document.getElementById("chat-status").getAttribute("data-state") === "error");
    check(await tab.textContent("#chat-status .sw-status__text") === "That question has gone.", `says why: ${await tab.textContent("#chat-status")}`);
    check(await tab.locator("[aria-disabled=true]").count() === 0, "the buttons are back");
    check(await tab.locator(".sw-proposal").count() === 1, "the question stays");
    await tab.close();
  },

  "proposal: one asked at the end of a turn is put in place, said, and answered in place": async (browser, check) => {
    const posted = [];
    const tab = await open(browser, page(chat("")), {
      "/proposal/p2/dismiss": (route, url) => { posted.push(url.pathname); return route.fulfill({ contentType: "text/html", body: page("<p>done</p>") }); },
    });
    await tab.evaluate((html) => { sw.refresh = () => {}; sw.emit("turn-done", { proposals: html }); }, proposal("p2"));
    check(await tab.locator("#p2").count() === 1, "the question is on the page");
    await tab.click("text=No, leave it");
    await tab.waitForFunction(() => !document.querySelector(".sw-proposal"));
    check(posted.join() === "/proposal/p2/dismiss", `answered in place: ${posted.join()}`);
    check(await tab.textContent("#chat-status .sw-status__text") === "Left as it was.", `said: ${await tab.textContent("#chat-status")}`);
    await tab.close();
  },

  "upload: sent with no file or an empty one, it says so in words; a picture asks what it shows": async (browser, check) => {
    const form = `<div class="sw-shell"><main id="main">` + readFileSync(new URL("../../design/components/upload/examples/default.html", import.meta.url), "utf8") + "</main></div>";
    const tab = await open(browser, page(form));
    const said = () => tab.textContent("#upload-error");
    await tab.click("text=Upload");
    check(/Select a file to add/.test(await said()), `no file: ${await said()}`);
    check(await tab.getAttribute("#upload", "aria-invalid") === "true", "the field is marked");
    await tab.setInputFiles("#upload", { name: "empty.txt", mimeType: "text/plain", buffer: Buffer.alloc(0) });
    check(/The selected file is empty/.test(await said()), `an empty file: ${await said()}`);
    check(await tab.isHidden(".sw-upload__about"), "what a picture shows is not asked of a text file");
    await tab.setInputFiles("#upload", { name: "p.png", mimeType: "image/png", buffer: Buffer.from([137, 80, 78, 71]) });
    check(await said() === "", `a file that will do: ${await said()}`);
    check(await tab.isVisible(".sw-upload__about"), "a picture asks what it shows");
    await tab.close();
  },
};
