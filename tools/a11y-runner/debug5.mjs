import { chromium } from "playwright";
import AxeBuilder from "@axe-core/playwright";
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";

const root = resolve(process.cwd(), "..", "..");
const componentsDir = join(root, "design", "components");

// Read shell.mjs functions directly here for debugging
const tokensPath = join(root, "design", "tokens", "tokens.css");
const tokens = readFileSync(tokensPath, "utf8");
const baseFiles = readdirSync(join(root, "design", "base"))
  .filter((f) => f.endsWith(".css"))
  .sort()
  .map((f) => readFileSync(join(root, "design", "base", f), "utf8"))
  .join("\n");

function shell(componentCss, body) {
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Example</title>
<style>${tokens}\n${base}\n${componentCss}</style></head>
<body><main><div class="shell"><h1>Example</h1><h2>Section</h2></div>${body}</main></body></html>`;
}

const AA_TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa", "best-practice"];

const browser = await chromium.launch();
const ctx = await browser.newContext();
const tab = await ctx.newPage();
let failures = 0;
let warnings = 0;

for (const name of readdirSync(componentsDir).sort()) {
  const dir = join(componentsDir, name);
  const examplesDir = join(dir, "examples");
  if (!existsSync(examplesDir)) continue;
  const css = existsSync(join(dir, "style.css")) ? readFileSync(join(dir, "style.css"), "utf8") : "";
  
  for (const file of readdirSync(examplesDir).filter((f) => f.endsWith(".html")).sort()) {
    const body = readFileSync(join(examplesDir, file), "utf8");
    
    // Use <form> wrapper exactly as run.mjs does
    await tab.setContent(shell(css, `<form>${body}</form>`));
    await new Promise(r => setTimeout(r, 300));

    console.log(`Testing ${name}/${file}...`);
    
    const aa = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
    for (const v of aa.violations) {
      failures++;
      console.log(`FAIL ${name}/${file}: ${v.id} - ${v.help} (${v.nodes.length} node(s))`);
      for (const node of v.nodes) {
        const sel = node.target.length > 0 ? node.target[0] : "?";
        const diag = await tab.evaluate((s) => {
          const el = document.querySelector(s);
          if (!el) return null;
          const cs = getComputedStyle(el);
          return { color: cs.color, backgroundColor: cs.backgroundColor };
        }, sel);
        console.log(`  Node ${sel}: fg=${diag?.color} bg=${diag?.backgroundColor}`);
      }
    }
    
    if (name === "disclosure") {
      const count = aa.violations.length;
      console.log(`  -> disclosure/${file}: ${count} violation(s)`);
    }
  }
}

await browser.close();
console.log(`a11y: ${failures} AA violation(s), ${warnings} AAA warning(s)`);
process.exit(failures > 0 ? 1 : 0);
