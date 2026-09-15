import { chromium } from 'playwright';
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

const browser = await chromium.launch();
const ctx = await browser.newContext();
const tab = await ctx.newPage();

// Test just disclosure with open.html (the failing case)
const discCssFile = path.join(root, 'design', 'components', 'disclosure', 'style.css');
const discCss = fs.readFileSync(discCssFile, 'utf8');
const body = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'open.html'), 'utf8');

// First test WITHOUT form wrapper
await tab.setContent(buildPage('', `<form>${body}</form>`));
await new Promise(r => setTimeout(r, 300));

// Dump all elements and their computed styles
const allEls = await tab.evaluate(() => {
  const results = [];
  document.querySelectorAll('*').forEach(el => {
    const cs = getComputedStyle(el);
    if (cs.display !== 'none' && el.innerText?.trim()) {
      // Check if this element has text that would be visible
      const parentBg = el.parentElement ? getComputedStyle(el.parentElement).backgroundColor : null;
      results.push({
        tag: el.tagName.toLowerCase(),
        classes: Array.from(el.classList),
        color: cs.color,
        bg: cs.backgroundColor,
        display: cs.display,
        text: (el.innerText || '').substring(0, 30),
        parentBg: parentBg,
      });
    }
  });
  return results;
});

console.log('=== Elements in open.html ===');
allEls.forEach(e => console.log(`  ${e.tag}.${e.classes.join('.')}: color=${e.color} bg=${e.bg} text="${e.text}"`));

// Now check which elements axe-core would evaluate for contrast
const axeCheckElements = await tab.evaluate(() => {
  const results = [];
  document.querySelectorAll('*').forEach(el => {
    const cs = getComputedStyle(el);
    // Only consider elements that have visible text content and a non-transparent background or inherit one
    if (cs.display === 'none' || cs.visibility === 'hidden') return;
    const hasText = el.innerText?.trim().length > 0;
    if (!hasText) return;
    
    results.push({
      tag: el.tagName.toLowerCase(),
      classes: Array.from(el.classList),
      color: cs.color,
      bg: cs.backgroundColor,
      text: (el.innerText || '').substring(0, 30),
    });
  });
  return results;
});

console.log('\n=== Elements axe would check ===');
axeCheckElements.forEach(e => console.log(`  ${e.tag}.${e.classes.join('.')}: color=${e.color} bg=${e.bg}`));

await browser.close();
