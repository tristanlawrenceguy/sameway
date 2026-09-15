import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '..', '..');
const componentsDir = path.join(root, 'design', 'components');

// Read shared resources once (like shell.mjs does)
const tokens = fs.readFileSync(path.join(root, 'design', 'tokens', 'tokens.css'), 'utf8');
const baseContent = fs.readdirSync(path.join(root, 'design', 'base'))
  .filter((f) => f.endsWith('.css')).sort()
  .map((f) => fs.readFileSync(path.join(root, 'design', 'base', f), 'utf8'))
  .join('\n');

function buildPage(cssComponent, bodyHtml) {
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Example</title>` +
    `<style>${tokens}\n${baseContent}\n${cssComponent}</style></head>` +
    `<body><main><div class="shell"><h1>Example</h1><h2>Section</h2></div>${bodyHtml}</main></body></html>`;
}

const AA_TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa", "best-practice"];

// Use ONE browser instance, ONE context, ONE page (reused across ALL components)
// This matches run.mjs exactly
const browser = await chromium.launch();
const ctx = await browser.newContext();
const tab = await ctx.newPage();

let failures = 0;
for (const name of fs.readdirSync(componentsDir).sort()) {
  const dir = path.join(componentsDir, name);
  const examplesDir = path.join(dir, 'examples');
  if (!fs.existsSync(examplesDir)) continue;
  
  const cssFile = path.join(dir, 'style.css');
  const css = fs.existsSync(cssFile) ? fs.readFileSync(cssFile, 'utf8') : '';

  for (const file of fs.readdirSync(examplesDir).filter((f) => f.endsWith('.html')).sort()) {
    const body = fs.readFileSync(path.join(examplesDir, file), 'utf8');
    
    await tab.setContent(buildPage(css, `<form>${body}</form>`));
    
    const aa = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
    for (const v of aa.violations) {
      failures++;
      console.log(`FAIL ${name}/${file}: ${v.id} - ${v.help}`);
      for (const node of v.nodes) {
        const sel = node.target.length > 0 ? node.target[0] : '?';
        const diag = await tab.evaluate((s) => {
          const el = document.querySelector(s);
          if (!el) return null;
          const cs = getComputedStyle(el);
          return { color: cs.color, backgroundColor: cs.backgroundColor };
        }, sel);
        console.log(`  Node ${sel}: fg=${diag?.color} bg=${diag?.backgroundColor}`);
      }
    }
    
    if (name === 'disclosure') {
      console.log(`  -> disclosure/${file}: ${aa.violations.length} violation(s)`);
    }
  }
}

await browser.close();
console.log(`Total failures: ${failures}`);
process.exit(failures > 0 ? 1 : 0);
