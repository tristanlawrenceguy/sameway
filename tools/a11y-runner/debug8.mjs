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

// Find which component's CSS causes the disclosure/open.html to fail
for (const blocker of fs.readdirSync(componentsDir).sort()) {
  const blockDir = path.join(componentsDir, blocker);
  const examplesDir = path.join(blockDir, 'examples');
  if (!fs.existsSync(examplesDir)) continue;

  for (const blockFile of fs.readdirSync(examplesDir).filter((f) => f.endsWith('.html')).sort()) {
    // Get the blocker component's CSS
    const blockerCssFile = path.join(blockDir, 'style.css');
    const blockerCss = fs.existsSync(blockerCssFile) ? fs.readFileSync(blockerCssFile, 'utf8') : '';

    // Now test disclosure/open.html with THIS blocker's CSS loaded first
    const discCssFile = path.join(root, 'design', 'components', 'disclosure', 'style.css');
    const discCss = fs.readFileSync(discCssFile, 'utf8');
    const body = fs.readFileSync(path.join(root, 'design', 'components', 'disclosure', 'examples', 'open.html'), 'utf8');

    // Page has blocker CSS + base + disclosure CSS (same order as run.mjs processes components)
    const pageHtml = buildPage(blockerCss + discCss, `<form>${body}</form>`);

    const browser = await chromium.launch();
    const ctx = await browser.newContext();
    const tab = await ctx.newPage();
    
    await tab.setContent(pageHtml);
    const aa = await new AxeBuilder({ page: tab }).withTags(AA_TAGS).analyze();
    
    if (aa.violations.length > 0) {
      console.log(`FAIL when ${blocker}/${blockFile} CSS is loaded before disclosure:`);
      for (const v of aa.violations) {
        console.log(`  ${v.id}: ${v.help}`);
        for (const node of v.nodes) {
          const sel = node.target[0] || '?';
          console.log(`    Node: ${sel} HTML: ${(node.html||'').substring(0,200)}`);
        }
      }
    } else {
      // Only check if this is a component that could plausibly affect disclosure
      const compList = ['callout', 'link', 'status', 'avatar', 'button', 'card', 'code-block', 
                         'heading', 'inline-code', 'kbd', 'list', 'logo', 'nav', 'note', 
                         'page-heading', 'paragraph', 'pressable', 'prose', 'quote',
                         'section-heading', 'status', 'tab-list'];
      if (!compList.includes(blocker)) {
        // console.log(`PASS with ${blocker}/${blockFile}`);
      }
    }

    await browser.close();
  }
}
