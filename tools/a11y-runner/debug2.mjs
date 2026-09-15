import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';
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
const ctx = await browser.newContext();
const page = await ctx.newPage();

await page.setContent('<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Example</title><style>' + tokens + '\n' + base + '\n' + css + '</style></head><body><main><div class="shell"><h1>Example</h1><h2>Section</h2></div>' + body + '</main></body></html>');
await page.waitForTimeout(500);

const aaTags = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa", "best-practice"];
const result = await new AxeBuilder({ page: page }).withTags(aaTags).analyze();

if (result.violations.length === 0) {
  console.log('No AA violations');
} else {
  for (const v of result.violations) {
    console.log(`VIOLATION: ${v.id} - ${v.help}`);
    for (const node of v.nodes) {
      const selector = node.target[0] || '?';
      const htmlSnippet = node.html?.substring(0, 200) || '';
      console.log(`  Node: ${selector}`);
      console.log(`  HTML: ${htmlSnippet}`);
      
      // Get computed styles for this specific node
      const elStyles = await page.evaluate((sel) => {
        const el = document.querySelector(sel);
        if (!el) return null;
        const cs = getComputedStyle(el);
        const parentEl = el.parentElement;
        const pcs = parentEl ? getComputedStyle(parentEl) : null;
        return { 
          color: cs.color, 
          backgroundColor: cs.backgroundColor, 
          display: cs.display,
          visibility: cs.visibility,
          opacity: cs.opacity,
          parentColor: pcs ? pcs.color : null,
          parentBg: pcs ? pcs.backgroundColor : null,
        };
      }, selector);
      console.log(`  Computed styles: ${JSON.stringify(elStyles, null, 4)}`);
    }
  }
}

await browser.close();
