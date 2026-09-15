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

// Test ONLY disclosure/open.html with its own CSS, single browser instance
const discCssFile = path.join(root, 'design', 'components', 'disclosure', 'style.css');
const discCss = fs.readFileSync(discCssFile, 'utf8');
const body = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'open.html'), 'utf8');

for (let i = 0; i < 5; i++) {
  const browser = await chromium.launch();
  const ctx = await browser.newContext();
  const tab = await ctx.newPage();
  
  await tab.setContent(buildPage(discCss, `<form>${body}</form>`));
  await new Promise(r => setTimeout(r, 100));

  // Check computed styles first
  const bodyStyle = await tab.evaluate(() => {
    const el = document.querySelector('.sw-disclosure__body');
    if (!el) return null;
    const cs = getComputedStyle(el);
    return { 
      color: cs.color, 
      backgroundColor: cs.backgroundColor,
      parentBg: el.parentElement ? getComputedStyle(el.parentElement).backgroundColor : 'no-parent',
    };
  });

  // Now run axe-core
  const aa = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
  
  console.log(`Run ${i+1}: bodyStyle=${JSON.stringify(bodyStyle)}, violations=${aa.violations.length}`);
  if (aa.violations.length > 0) {
    for (const v of aa.violations) {
      for (const node of v.nodes) {
        const sel = node.target[0] || '?';
        console.log(`  FAIL: ${sel} - ${(node.html||'').substring(0,200)}`);
      }
    }
  }

  await browser.close();
}
