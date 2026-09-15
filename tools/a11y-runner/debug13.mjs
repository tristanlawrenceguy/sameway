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

// Test different variants of open.html
const discCss = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'style.css'), 'utf8');

const tests = [
  { name: 'open with animation CSS', html: `<details class="sw-disclosure" data-component="disclosure" open><summary class="sw-disclosure__summary sw-pressable"><span class="sw-disclosure__label">Raw props</span></summary><div class="sw-disclosure__body">{&#34;text&#34;:&#34;Shopping&#34;}</div></details>` },
  { name: 'open without animation CSS', html: `<details class="sw-disclosure" data-component="disclosure" open><summary class="sw-disclosure__summary sw-pressable"><span class="sw-disclosure__label">Raw props</span></summary><div class="sw-disclosure__body">{&#34;text&#34;:&#34;Shopping&#34;}</div></details>` },
  { name: 'closed (no open attr)', html: `<details class="sw-disclosure" data-component="disclosure"><summary class="sw-disclosure__summary sw-pressable"><span class="sw-disclosure__label">Raw props</span></summary><div class="sw-disclosure__body">{&#34;text&#34;:&#34;Shopping&#34;}</div></details>` },
];

for (const test of tests) {
  const browser = await chromium.launch();
  const ctx = await browser.newContext();
  const tab = await ctx.newPage();
  
  // Build page with all CSS including animation
  await tab.setContent(buildPage(discCss, `<form>${test.html}</form>`));
  await new Promise(r => setTimeout(r, 100));

  const bodyStyle = await tab.evaluate(() => {
    const el = document.querySelector('.sw-disclosure__body');
    if (!el) return null;
    const cs = getComputedStyle(el);
    return { 
      color: cs.color, 
      backgroundColor: cs.backgroundColor,
      display: cs.display,
      animationName: cs.animationName,
    };
  });

  // Check body visibility
  const isVisible = await tab.evaluate(() => {
    const el = document.querySelector('.sw-disclosure__body');
    if (!el) return false;
    const cs = getComputedStyle(el);
    return cs.display !== 'none' && cs.visibility !== 'hidden';
  });

  const aa = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
  
  console.log(`${test.name}: visible=${isVisible}, style=${JSON.stringify(bodyStyle)}, violations=${aa.violations.length}`);
  if (aa.violations.length > 0) {
    for (const v of aa.violations) {
      for (const node of v.nodes) {
        console.log(`  Node: ${(node.html||'').substring(0,200)}`);
      }
    }
  }

  await browser.close();
}
