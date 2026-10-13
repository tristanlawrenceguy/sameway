// What the stylesheet does to what a person sees, read from the page as
// drawn: the components' own examples (their golden HTML) in a page with
// /design/sameway.css, the way the server sends them.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { open, page } from "./harness.mjs";

const example = (component, name) => readFileSync(fileURLToPath(new URL(`../../design/components/${component}/examples/${name}.html`, import.meta.url)), "utf8");

const login = `<span class="sw-person" data-component="person" data-person="2"><span class="sw-person__dot" aria-hidden="true"></span><bdi>maximiliane.wolkonskaja.fitzgerald@example.com</bdi></span>`;

export const cases = {
  "look: a field's error is shown as its words alone; Error: is said, not shown": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main">${example("text-field", "error")}</main>`));
    const err = await tab.evaluate(() => {
      const p = document.querySelector(".sw-field__error");
      // What is drawn: text in a box of a pixel or less is for screen readers.
      const shown = [...p.childNodes].filter((n) => n.nodeType === 3 || n.getBoundingClientRect().width > 1).map((n) => n.textContent).join("").trim();
      return { before: getComputedStyle(p, "::before").content, shown };
    });
    check(["none", "normal"].includes(err.before), `nothing is put before the message: ${err.before}`);
    check(err.shown === "Enter an email address like name@example.com.", `a person sees the message only: ${JSON.stringify(err.shown)}`);
    await tab.close();
  },

  // A login stands in for a name nobody gave: no spaces to break at.
  "look: a long name wraps on a phone, on its own and in a row's details": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main" style="width: 320px">${example("person", "long")}
<p>${login}</p><ul class="sw-rows"><li class="sw-row"><a href="/t/task/1">Water the plants</a><span class="sw-row__meta">${login}</span></li></ul></main>`));
    const over = await tab.evaluate(() => [...document.querySelectorAll(".sw-person")].map((p) => {
      const r = p.getBoundingClientRect(), m = document.getElementById("main").getBoundingClientRect();
      return Math.max(m.left - r.left, r.right - m.right); // past either edge
    }));
    check(over.length === 3 && over.every((d) => d <= 0.5), `each name stays inside the page's 320px: ${JSON.stringify(over)}`);
    await tab.close();
  },

  "look: a block set up wrong says so in the text colour, not muted like an empty line": async (browser, check) => {
    const tab = await open(browser, page(`<main id="main">${example("problem", "default")}${example("empty", "default")}</main>`));
    const colours = await tab.evaluate(() => ({ said: getComputedStyle(document.querySelector(".sw-problem__said")).color, empty: getComputedStyle(document.querySelector(".sw-empty__message")).color, text: getComputedStyle(document.body).color }));
    check(colours.said !== colours.empty && colours.said === colours.text, `the problem is in the text colour, the empty line muted: ${JSON.stringify(colours)}`);
    await tab.close();
  },
};
