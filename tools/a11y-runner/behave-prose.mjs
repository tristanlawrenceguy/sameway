// Rich text in an edit (design/base/14-prose-edit.js and 16-prose-tools.js):
// Tab reaches the words before anything that formats them, the toolbar is
// one Tab stop whose buttons the arrow keys move between, then the switch
// to Markdown, which shows one Body field at a time; a field of several
// lines is a box Tab reaches too.
import { open, page } from "./harness.mjs";

const controls = `<template id="sw-controls"><div data-control="text"><div class="sw-field"><label>Field</label><input type="text"></div></div><div data-control="textarea"><div class="sw-field"><label>Field</label><textarea></textarea></div></div></template>`;
const block = `<li class="sw-block" data-block-id="b1" data-block-component="note" data-block-label="Plan" data-edit-action="/t/note/n1/props">
<div class="sw-bar"></div><h3 data-prop="title">Plan</h3>
<div class="sw-prose" data-prop="body" data-source="Water the **plants**."><p>Water the <strong>plants</strong>.</p></div>
<p data-prop="notes">first line
second line</p></li>`;
const shell = page(`${controls}<div class="sw-shell"><main id="main" tabindex="-1"><ol class="sw-canvas">${block}</ol></main></div>`);

// focused says what has focus, in words a failure can show.
const focused = (tab) => tab.evaluate(() => {
  const a = document.activeElement;
  return a.classList.contains("sw-prose-editor") ? "Body" : a.closest(".sw-prose-foot [role=toolbar], .sw-prose-foot .sw-prose-tools") ? "tool:" + (a.getAttribute("aria-label") || a.textContent.trim())
    : a.tagName === "TEXTAREA" && a.name === "prop-body" ? "Markdown field" : a.tagName === "TEXTAREA" ? "textarea:" + a.name : a.textContent.trim() || a.name || a.tagName;
});

export const cases = {
  "prose: Tab reaches Body first, then the toolbar as one stop, then the switch to Markdown": async (browser, check) => {
    const tab = await open(browser, shell);
    await tab.getByRole("button", { name: "Edit Plan" }).click();
    check(await focused(tab) === "prop-title", `the edit opens on its first field: ${await focused(tab)}`);
    await tab.keyboard.press("Tab");
    check(await focused(tab) === "Body", `Tab from the title reaches Body: ${await focused(tab)}`);
    await tab.keyboard.press("Tab");
    const first = await focused(tab);
    check(first.startsWith("tool:"), `then the toolbar: ${first}`);
    await tab.keyboard.press("Tab");
    check(await focused(tab) === "Markdown", `one Tab passes the whole toolbar: ${await focused(tab)}`);
    await tab.keyboard.press("Shift+Tab");
    await tab.keyboard.press("ArrowRight");
    const second = await focused(tab);
    check(second.startsWith("tool:") && second !== first, `the arrow keys move along it: ${first} then ${second}`);
    await tab.keyboard.press("Tab");
    check(await focused(tab) === "Markdown", `one Tab leaves the toolbar for the switch to Markdown: ${await focused(tab)}`);
    await tab.keyboard.press("Shift+Tab");
    check(await focused(tab) === second, `Shift+Tab comes back to the button last used: ${await focused(tab)}`);
    await tab.close();
  },

  "prose: Markdown shows one Body field at a time, and switches back": async (browser, check) => {
    const tab = await open(browser, shell, {
      "/api/prose": (route, url, req) => route.fulfill({ contentType: "application/json", body: JSON.stringify(JSON.parse(req.postData()).html ? { markdown: "Water the **plants**." } : { html: "<p>Water the <strong>plants</strong>.</p>" }) }),
    });
    await tab.getByRole("button", { name: "Edit Plan" }).click();
    const bodies = () => tab.evaluate(() => [...document.querySelectorAll(".sw-prose-field .sw-prose-editor, .sw-prose-field textarea")].filter((e) => !e.hidden && e.getClientRects().length).map((e) => e.tagName));
    check((await bodies()).join() === "DIV", `at first, the rich text alone: ${await bodies()}`);
    await tab.getByRole("button", { name: "Markdown" }).click();
    await tab.waitForFunction(() => document.activeElement.name === "prop-body");
    check((await bodies()).join() === "TEXTAREA", `then the Markdown alone, with focus in it: ${await bodies()}`);
    check(await tab.inputValue("textarea[name=prop-body]") === "Water the **plants**.", "holding the words");
    await tab.getByRole("button", { name: "Markdown" }).click();
    await tab.waitForFunction(() => document.activeElement.classList.contains("sw-prose-editor"));
    check((await bodies()).join() === "DIV", `and back: ${await bodies()}`);
    await tab.close();
  },

  "prose: a field of several lines is a box Tab reaches": async (browser, check) => {
    const tab = await open(browser, shell);
    await tab.getByRole("button", { name: "Edit Plan" }).click();
    const seen = [];
    for (let i = 0; i < 12 && !seen.includes("textarea:prop-notes"); i++) {
      await tab.keyboard.press("Tab");
      seen.push(await focused(tab));
    }
    check(seen.includes("textarea:prop-notes"), `Tab reaches the notes: ${seen.join(", ")}`);
    await tab.close();
  },
};
