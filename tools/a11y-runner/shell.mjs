// Shared page shell for rendering a component example in isolation. It
// supplies the same tokens and base CSS the real server serves, plus an h1
// and h2 so components that default to heading level 3 sit in a valid outline.
import { readFileSync } from "node:fs";
import { join, resolve } from "node:path";

const root = resolve(process.cwd(), "..", "..");
const designDir = join(root, "design");
// axe-core rule tags: AA (plus best practices) fails a run, AAA only warns.
export const AA_TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa", "best-practice"];
export const AAA_TAGS = ["wcag2aaa", "wcag21aaa", "wcag22aaa"];

export const tokens = readFileSync(join(designDir, "tokens", "tokens.css"), "utf8");
export const base = readFileSync(join(designDir, "base", "base.css"), "utf8");

export function shell(componentCss, body) {
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Example</title>
<style>${tokens}\n${base}\n${componentCss}</style></head>
<body><main><div class="shell"><h1>Example</h1><h2>Section</h2></div>${body}</main></body></html>`;
}
