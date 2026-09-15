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

// Test both with and without <form> wrapper
for (const wrapper of ['<form>', '']) {
  console.log(`=== Testing with ${wrapper} wrapper ===`);
  
  const browser = await chromium.launch();
  const ctx = await browser.newContext();
  const page = await ctx.newPage();

  for (const exampleFile of fs.readdirSync(path.join(root, 'design', 'components', 'disclosure', 'examples')).filter(f => f.endsWith('.html')).sort()) {
    const css = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'style.css'), 'utf8');
    const body = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', exampleFile), 'utf8');

    await page.setContent('<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Example</title><style>' + tokens + '\n' + base + '\n' + css + '</style></head><body><main><div class="shell"><h1>Example</h1><h2>Section</h2></div>' + wrapper + body + '</form></main></body></html>');
    await page.waitForTimeout(500);

    const aaTags = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa", "best-practice"];
    const result = await new AxeBuilder({ page: page }).withTags(aaTags).analyze();

    if (result.violations.length > 0) {
      for (const v of result.violations) {
        console.log(`  FAIL ${exampleFile}: ${v.id} - ${v.help}`);
        for (const node of v.nodes) {
          const selector = node.target[0] || '?';
          const elStyles = await page.evaluate((sel) => {
            const el = document.querySelector(sel);
            if (!el) return null;
            const cs = getComputedStyle(el);
            return { 
              color: cs.color, 
              backgroundColor: cs.backgroundColor, 
              display: cs.display,
            };
          }, selector);
          console.log(`    Node ${selector}: styles=${JSON.stringify(elStyles)}`);
        }
      }
    } else {
      console.log(`  PASS ${exampleFile}: no violations`);
    }
  }

  await browser.close();
}
