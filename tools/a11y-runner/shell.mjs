// Shared page shell for rendering a component example in isolation. It
// supplies the same tokens, base and component CSS the real server serves,
// plus an h1 and h2 so components that default to heading level 3 sit in a
// valid outline, inside .sw-shell and main.sw-main so they get the column
// and gutter a real page gives them.
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { join, resolve } from "node:path";

const __dirname = fileURLToPath(new URL(".", import.meta.url));
const root = resolve(__dirname, "..", "..");
const designDir = join(root, "design");
// axe-core rule tags: AA (plus best practices) fails a run, and so does AAA.
export const AA_TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa", "best-practice"];
export const AAA_TAGS = ["wcag2aaa", "wcag21aaa", "wcag22aaa"];

export const tokens = readFileSync(join(designDir, "tokens", "tokens.css"), "utf8");
// Every stylesheet in design/base, in filename order, exactly as the server
// concatenates them.
export const base = readdirSync(join(designDir, "base"))
  .filter((f) => f.endsWith(".css"))
  .sort()
  .map((f) => readFileSync(join(designDir, "base", f), "utf8"))
  .join("\n");

// Every component's stylesheet in name order, as /design/sameway.css has
// them: an example uses other components (a proposal's buttons, a
// calendar's links), and they need their own styles too.
const componentsDir = join(designDir, "components");
export const components = readdirSync(componentsDir)
  .sort()
  .filter((name) => existsSync(join(componentsDir, name, "style.css")))
  .map((name) => readFileSync(join(componentsDir, name, "style.css"), "utf8"))
  .join("\n");

export function shell(body) {
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Example</title>
<style>${tokens}\n${base}\n${components}</style></head>
<body><div class="sw-shell"><main class="sw-main"><div class="shell"><h1>Example</h1><h2>Section</h2></div>${body}</main></div></body></html>`;
}
