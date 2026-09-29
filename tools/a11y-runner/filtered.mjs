// filtered.mjs: pages that narrow a list with the filters component, in
// both shapes: a calendar of everything with two kinds this month,
// narrowed by kind (links), and the activity log narrowed (a form). Each
// offers its filters and passes axe AA and AAA; the paths are returned so
// pages.mjs checks them as agents see them too.
import { axeProblems } from "./checks.mjs";
import { AA_TAGS, AAA_TAGS } from "./shell.mjs";

export async function filtered(page, base, check, fail) {
  const post = (path, body) => fetch(base + path, { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(body) });
  const today = new Date().toISOString().slice(0, 10) + "T00:00:00Z";
  await post("/api/task", { title: "Kind check task", due: today });
  await post("/api/reminder", { title: "Kind check reminder", at: today });
  const cal = await (await post("/api/block", { component: "calendar", props: { type: "all", detail: "page", caption: "Everything" } })).json();
  const paths = ["/activity?who=you&when=week", "/activity?kind=task", `/canvas/${cal.id}`, `/canvas/${cal.id}?c-${cal.id}-type=task`];
  for (const path of paths) {
    await page.goto(base + path);
    check(await page.locator('[data-component="filters"]').count() > 0, `${path}: offers its filters`);
    for (const p of await axeProblems(page, [], [...AA_TAGS, ...AAA_TAGS])) fail(`${path}: ${p}`);
  }
  return paths;
}
