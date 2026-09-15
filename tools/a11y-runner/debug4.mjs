import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '..', '..');
const componentsDir = path.join(root, 'design', 'components');

// Run exactly like run.mjs: iterate all components in sort order
for (const name of fs.readdirSync(componentsDir).sort()) {
  const dir = path.join(componentsDir, name);
  const examplesDir = path.join(dir, 'examples');
  if (!fs.existsSync(examplesDir)) continue;
  
  const cssFile = path.join(dir, 'style.css');
  const css = fs.existsSync(cssFile) ? fs.readFileSync(cssFile, 'utf8') : '';

  for (const file of fs.readdirSync(examplesDir).filter(f => f.endsWith('.html')).sort()) {
    const body = fs.readFileSync(path.join(examplesDir, file), 'utf8');
    
    // Build the page exactly as run.mjs does
    const tokens = fs.readFileSync(path.join(root, 'design', 'tokens', 'tokens.css'), 'utf8');
    const baseFiles = fs.readdirSync(path.join(root, 'design', 'base')).filter(f => f.endsWith('.css')).sort();
    const base = baseFiles.map(f => fs.readFileSync(path.join(root, 'design', 'base', f), 'utf8')).join('\n');
    
    const html = '<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Example</title>' +
      '<style>' + tokens + '\n' + base + '\n' + css + '</style></head>' +
      '<body><main><div class="shell"><h1>Example</h1><h2>Section</h2></div>' +
      '<form>' + body + '</form></main></body></html>';

    // Use the same context and page (reusing like run.mjs does)
    const browser = await chromium.launch();
    const ctx = await browser.newContext();
    const tab = await ctx.newPage();

    await tab.setContent(html);
    await new Promise(r => setTimeout(r, 300));

    const aaTags = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa", "best-practice"];
    const result = await new AxeBuilder({ page: tab }).withTags(aaTags).analyze();

    if (result.violations.length > 0) {
      for (const v of result.violations) {
        console.log(`FAIL ${name}/${file}: ${v.id} - ${v.help}`);
      }
    } else {
      // Only print disclosure to keep output manageable
      if (name === 'disclosure') {
        console.log(`PASS disclosure/${file}: no violations`);
      }
    }

    await browser.close();
  }
}
