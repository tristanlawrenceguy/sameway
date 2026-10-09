// Editing in place (design/base/09-edit.js): the Edit button is named for
// its block, the form goes to the block's own edit address, what was read
// and the block's bar step aside while it is open, and Cancel or Escape
// puts them back with focus on Edit. A link to a place on the page is left
// to the browser.
import { open, page } from "./harness.mjs";

const controls = `<template id="sw-controls"><div data-control="text"><div class="sw-field"><label>Field</label><input type="text"></div></div></template>`;
const block = (attrs) => `<li class="sw-block" data-block-id="b1" data-block-component="note" ${attrs}>
<div class="sw-bar"><a href="/t/note/n1">Open</a></div>
<h3 data-prop="title">Shopping</h3><dl class="sw-fields"><dt>Tags</dt><dd>home</dd></dl></li>`;
const shell = (b) => page(`${controls}<a class="sw-skip" href="#main">Skip to content</a><div class="sw-shell"><main id="main" tabindex="-1"><ol class="sw-canvas">${b}</ol></main></div>`);

export const cases = {
  "edit: named for its block, sent to its own address, and put back by Cancel with focus on Edit": async (browser, check) => {
    const tab = await open(browser, shell(block(`data-block-label="Shopping" data-edit-action="/t/note/n1/props"`)));
    const edit = tab.getByRole("button", { name: "Edit Shopping" });
    check(await edit.count() === 1, "the Edit button says which block");
    await edit.click();
    const form = await tab.evaluate(() => {
      const f = document.querySelector(".sw-inline-form");
      return f && { action: new URL(f.action).pathname, focused: document.activeElement.name, dl: document.querySelector("dl.sw-fields").hidden, bar: getComputedStyle(document.querySelector(".sw-bar")).display };
    });
    check(form && form.action === "/t/note/n1/props", `the form goes to the block's own address: ${form && form.action}`);
    check(form && form.focused === "prop-title", `focus is in the first field: ${form && form.focused}`);
    check(form && form.dl === true && form.bar === "none", `what was read and the bar step aside: ${JSON.stringify(form)}`);
    await tab.getByRole("button", { name: "Cancel" }).click();
    const back = await tab.evaluate(() => ({ form: !!document.querySelector(".sw-inline-form"), dl: document.querySelector("dl.sw-fields").hidden, bar: getComputedStyle(document.querySelector(".sw-bar")).display, focus: document.activeElement.hasAttribute("data-edit") }));
    check(!back.form && !back.dl && back.bar !== "none", `Cancel puts it all back: ${JSON.stringify(back)}`);
    check(back.focus, "focus is back on Edit");
    await edit.click();
    await tab.keyboard.press("Escape");
    check(await tab.locator(".sw-inline-form").count() === 0, "Escape puts it back too");
    await tab.close();
  },

  "edit: a canvas block with no address of its own is saved on the canvas, and named by its kind": async (browser, check) => {
    const tab = await open(browser, shell(block("")));
    const edit = tab.getByRole("button", { name: "Edit note" });
    check(await edit.count() === 1, "named by its kind when it has no label");
    await edit.click();
    check(await tab.evaluate(() => new URL(document.querySelector(".sw-inline-form").action).pathname) === "/canvas/b1/props", "saved on the canvas");
    await tab.close();
  },

  "edit: a link to a place on the page is the browser's": async (browser, check) => {
    const tab = await open(browser, shell(block(`data-block-label="Shopping"`)));
    await tab.click("text=Skip to content");
    check(await tab.evaluate(() => location.hash) === "#main", "the skip link goes where it says");
    await tab.close();
  },
};
