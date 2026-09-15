import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '..', '..');
const componentsDir = path.join(root, 'design', 'components');

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

// Test: render each component that comes before disclosure, then check disclosure/open.html
for (const name of ['alert', 'badge', 'button', 'calendar', 'card', 'chat', 'checkbox', 'datepicker']) {
  const dir = path.join(componentsDir, name);
  const examplesDir = path.join(dir, 'examples');
  if (!fs.existsSync(examplesDir)) continue;

  for (const file of fs.readdirSync(examplesDir).filter((f) => f.endsWith('.html')).sort()) {
    const cssFile = path.join(dir, 'style.css');
    const css = fs.readFileSync(cssFile, 'utf8');
    const body = fs.readFileSync(path.join(examplesDir, file), 'utf8');

    // Render this component first
    const browser = await chromium.launch();
    const ctx = await browser.newContext();
    const tab = await ctx.newPage();
    
    await tab.setContent(buildPage(css, `<form>${body}</form>`));
    
    // Then render disclosure/open.html
    const discCss = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'style.css'), 'utf8');
    const openBody = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'open.html'), 'utf8');
    
    await tab.setContent(buildPage(discCss, `<form>${openBody}</form>`));

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
    
    if (aa.violations.length > 0) {
      console.log(`FAIL after ${name}/${file}: style=${JSON.stringify(bodyStyle)}, violations=${aa.violations.length}`);
    } else {
      // Only print non-passing ones... but we need to find the failing one. Print all for now.
      // Actually, let me just print failures. But since debug9 showed it fails even after default.html (which is disclosure itself), 
      // this test might not reproduce it because we're starting fresh each time.
    }

    await browser.close();
  }
}
