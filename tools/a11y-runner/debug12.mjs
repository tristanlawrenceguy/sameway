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

// Test: process all components in order, then check disclosure/open.html after each one
const discCssFile = path.join(root, 'design', 'components', 'disclosure', 'style.css');
const discCss = fs.readFileSync(discCssFile, 'utf8');


// Read the disclosure body content
const discBodyContent = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'open.html'), 'utf8');

const browser = await chromium.launch();
const ctx = await browser.newContext();
const tab = await ctx.newPage();

// First render disclosure/default.html (this passes)
const defCssFile = path.join(root, 'design', 'components', 'disclosure', 'style.css');
const defBody = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'default.html'), 'utf8');

await tab.setContent(buildPage(fs.readFileSync(defCssFile, 'utf8'), `<form>${defBody}</form>`));
let aa1 = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
console.log(`After default.html: ${aa1.violations.length} violations`);

// Now render disclosure/open.html (this fails)
await tab.setContent(buildPage(discCss, `<form>${discBodyContent}</form>`));
let aa2 = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
console.log(`After open.html: ${aa2.violations.length} violations`);

// Now let's try rendering each other component and then re-rendering open.html
for (const name of fs.readdirSync(componentsDir).sort()) {
  if (name === 'disclosure') continue; // skip disclosure itself
  
  const dir = path.join(componentsDir, name);
  const examplesDir = path.join(dir, 'examples');
  if (!fs.existsSync(examplesDir)) continue;

  for (const file of fs.readdirSync(examplesDir).filter((f) => f.endsWith('.html')).sort()) {
    const cssFile = path.join(dir, 'style.css');
    const css = fs.existsSync(cssFile) ? fs.readFileSync(cssFile, 'utf8') : '';
    const body = fs.readFileSync(path.join(examplesDir, file), 'utf8');

    // Render this component
    await tab.setContent(buildPage(css, `<form>${body}</form>`));
    
    // Then render disclosure/open.html
    await tab.setContent(buildPage(discCss, `<form>${discBodyContent}</form>`));
    
    const aa = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
    
    if (aa.violations.length > 0) {
      console.log(`VIOLATION after ${name}/${file}:`);
      for (const v of aa.violations) {
        console.log(`  ${v.id}: ${v.help}`);
        for (const node of v.nodes) {
          const sel = node.target[0] || '?';
          console.log(`    Node: ${sel}`);
        }
      }
    } else {
      // Only print if it's something interesting
      // console.log(`OK after ${name}/${file}`);
    }
  }
}

await browser.close();
