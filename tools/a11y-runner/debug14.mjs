import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '..', '..');

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

const discCss = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'style.css'), 'utf8');
const defBody = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'default.html'), 'utf8');
const openBody = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'open.html'), 'utf8');

// Test: reuse page, render default then open, with varying wait times
for (const [label, waits] of [
  ['no wait', [0, 0]],
  ['10ms wait', [0, 10]],
  ['50ms wait', [0, 50]],
  ['100ms wait', [0, 100]],
  ['200ms wait', [0, 200]],
]) {
  const browser = await chromium.launch();
  const ctx = await browser.newContext();
  const tab = await ctx.newPage();
  
  // Render default.html (passes)
  await tab.setContent(buildPage(discCss, `<form>${defBody}</form>`));
  if (waits[0] > 0) await new Promise(r => setTimeout(r, waits[0]));

  // Render open.html (should fail in the original test)
  await tab.setContent(buildPage(discCss, `<form>${openBody}</form>`));
  if (waits[1] > 0) await new Promise(r => setTimeout(r, waits[1]));

  const bodyStyle = await tab.evaluate(() => {
    const el = document.querySelector('.sw-disclosure__body');
    if (!el) return null;
    const cs = getComputedStyle(el);
    return { 
      color: cs.color, 
      backgroundColor: cs.backgroundColor,
      display: cs.display,
    };
  });

  const aa = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
  
  console.log(`${label}: style=${JSON.stringify(bodyStyle)}, violations=${aa.violations.length}`);

  await browser.close();
}
