// Component scripts moved out of design/base into the component's own
// enhance.js (AGENTS.md): what each promises, unchanged by the move.
import { open, page } from "./harness.mjs";

const alert = `<div class="sw-alert sw-alert--success sw-alert--dismissible" data-component="alert" data-kind="success"><p class="sw-alert__message">Changes made.</p><button type="button" class="sw-alert__close" data-dismiss aria-label="Close message"><span aria-hidden="true">×</span></button></div>`;

export const cases = {
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
