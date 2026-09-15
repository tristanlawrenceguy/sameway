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

// Replicate debug9 exactly: single reused page, render default then open
const discCss = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'style.css'), 'utf8');
const defBody = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'default.html'), 'utf8');
const openBody = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'open.html'), 'utf8');

// Also render components that come before disclosure to match run.mjs order
const componentsDir = path.join(root, 'design', 'components');
for (const name of fs.readdirSync(componentsDir).sort()) {
  if (name === 'disclosure') continue; // handle separately
  
  const dir = path.join(componentsDir, name);
  const examplesDir = path.join(dir, 'examples');
  if (!fs.existsSync(examplesDir)) continue;

  for (const file of fs.readdirSync(examplesDir).filter((f) => f.endsWith('.html')).sort()) {
    const cssFile = path.join(dir, 'style.css');
    const css = fs.readFileSync(cssFile, 'utf8');
    const body = fs.readFileSync(path.join(examplesDir, file), 'utf8');

    const browser = await chromium.launch();
    const ctx = await browser.newContext();
    const tab = await ctx.newPage();

    // Render all prior components first (to build up state)
    for (const prevName of fs.readdirSync(componentsDir).sort()) {
      if (prevName === 'disclosure') continue;
      
      const prevDir = path.join(componentsDir, prevName);
      const prevExamples = path.join(prevDir, 'examples');
      if (!fs.existsSync(prevExamples)) continue;

      for (const pf of fs.readdirSync(prevExamples).filter((f) => f.endsWith('.html')).sort()) {
        const pcssFile = path.join(prevDir, 'style.css');
        const pcss = fs.readFileSync(pcssFile, 'utf8');
        const pbody = fs.readFileSync(path.join(prevExamples, pf), 'utf8');

        await tab.setContent(buildPage(pcss, `<form>${pbody}</form>`));
      }
    }

    // Now render disclosure/default.html (passes)
    await tab.setContent(buildPage(discCss, `<form>${defBody}</form>`));
    
    const bodyStyleDefault = await tab.evaluate(() => {
      const el = document.querySelector('.sw-disclosure__body');
      if (!el) return null;
      const cs = getComputedStyle(el);
      return { color: cs.color, backgroundColor: cs.backgroundColor };
    });

    // Now render disclosure/open.html (should fail!)
    await tab.setContent(buildPage(discCss, `<form>${openBody}</form>`));
    
    const bodyStyleOpen = await tab.evaluate(() => {
      const el = document.querySelector('.sw-disclosure__body');
      if (!el) return null;
      const cs = getComputedStyle(el);
      return { color: cs.color, backgroundColor: cs.backgroundColor };
    });

    const aa = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
    
    console.log(`default.html style=${JSON.stringify(bodyStyleDefault)}, open.html style=${JSON.stringify(bodyStyleOpen)}, violations=${aa.violations.length}`);
    if (aa.violations.length > 0) {
      for (const v of aa.violations) {
        for (const node of v.nodes) {
          console.log(`  FAIL: ${(node.html||'').substring(0,200)}`);
        }
      }
    }

    await browser.close();
  }
}
