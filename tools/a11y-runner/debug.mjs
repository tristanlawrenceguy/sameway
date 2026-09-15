import { chromium } from 'playwright';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '..', '..');
const tokens = fs.readFileSync(path.join(root, 'design', 'tokens', 'tokens.css'), 'utf8');
const baseFiles = fs.readdirSync(path.join(root, 'design', 'base')).filter(f => f.endsWith('.css')).sort();
const base = baseFiles.map(f => fs.readFileSync(path.join(root, 'design', 'base', f), 'utf8')).join('\n');
const css = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'style.css'), 'utf8');
const body = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'open.html'), 'utf8');

const browser = await chromium.launch();
const page = await browser.newPage();

await page.setContent('<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Example</title><style>' + tokens + '\n' + base + '\n' + css + '</style></head><body><main><div class="shell"><h1>Example</h1><h2>Section</h2></div>' + body + '</main></body></html>');
await page.waitForTimeout(500);

const styles = await page.evaluate(() => {
  const el = document.querySelector('.sw-disclosure__body');
  if (!el) return null;
  const cs = getComputedStyle(el);
  const parentEl = el.parentElement;
  const pcs = parentEl ? getComputedStyle(parentEl) : null;
  return { 
    color: cs.color, 
    backgroundColor: cs.backgroundColor, 
    bg: cs.background,
    display: cs.display,
    parentColor: pcs ? pcs.color : null,
    parentBg: pcs ? pcs.backgroundColor : null,
    parentDisplay: pcs ? pcs.display : null,
  };
});
console.log('Body styles:', JSON.stringify(styles, null, 2));

// Get all elements with their computed fg/bg that axe might check
const allElements = await page.evaluate(() => {
  const results = [];
  document.querySelectorAll('*').forEach(el => {
    const cs = getComputedStyle(el);
    if (cs.color && cs.color !== 'rgba(0, 0, 0, 0)' && cs.backgroundColor) {
      const text = el.innerText || '';
      if (text.trim().length > 0) {
        results.push({
          tag: el.tagName.toLowerCase(),
          classes: Array.from(el.classList).join(' '),
          color: cs.color,
          backgroundColor: cs.backgroundColor,
          display: cs.display,
          text: text.substring(0, 50),
        });
      }
    }
  });
  return results;
});
console.log('\nAll visible elements with fg/bg:');
allElements.forEach(e => console.log(`  ${e.tag}.${e.classes}: color=${e.color} bg=${e.backgroundColor} display=${e.display}`));

await browser.close();
