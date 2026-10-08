// What the page's scripts do, in a real browser, with no server: each case
// is a small page carrying the scripts exactly as /design/sameway.js joins
// them (design/base in filename order, then each component's enhance.js
// in name order), served from a made-up origin whose answers the case
// gives (the page itself when it refreshes, a stream, a refusal). Cases
// live in behave-*.mjs, one file per part of the core (00-sw.js).
//
// These replace tests that read the scripts for exact strings: a string
// check passes on code that does not work and fails on a rename.
import { chromium } from "playwright";
import { readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";

const files = readdirSync(fileURLToPath(new URL(".", import.meta.url))).filter((f) => /^behave-.+\.mjs$/.test(f)).sort();
const only = process.argv[2];
const browser = await chromium.launch();
let failures = 0, ran = 0;
for (const f of files) {
  const { cases } = await import("./" + f);
  for (const [name, run] of Object.entries(cases)) {
    if (only && !name.includes(only)) continue;
    ran++;
    const fails = [];
    const check = (ok, msg) => { if (!ok) fails.push(msg); };
    try { await run(browser, check); } catch (e) { fails.push("threw: " + (e && e.stack || e)); }
    failures += fails.length;
    console.log(`${fails.length ? "FAIL" : "ok  "} ${f.replace(/^behave-|\.mjs$/g, "")}: ${name}`);
    fails.forEach((m) => console.log("     " + m));
  }
}
await browser.close();
console.log(`behave: ${ran} cases, ${failures} failures`);
process.exit(failures ? 1 : 0);
