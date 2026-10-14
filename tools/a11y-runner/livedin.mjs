// livedin.mjs: the workspace as people live in it, for the page checks.
// A fresh workspace has short titles and nobody else's changes, so two
// things a phone meets every day were never drawn in CI:
//   - words longer than a line: a pasted link, an email address, a long
//     name. Kept whole, one pushed the page sideways at 320px (1.4.10).
//   - "Since you were last here", the changes by others listed at the top
//     of every page once the person has been away. Its panel is narrower
//     than the log, and one change in it scrolled every page sideways on a
//     phone for two months (#619) while every check here passed.
// The records go in through the API, so they are an agent's changes, and
// the person is marked away from before they were made, the way the
// server marks them after half an hour: the one note it keeps for that,
// written in the workspace's database.
import { DatabaseSync } from "node:sqlite";
import { join } from "node:path";

export const LONG = {
  link: "Read https://example.com/reports/2026/quarterly-review-final-v2.pdf before Friday",
  name: "Alexandra Konstantinopoulou-Wolfeschlegelsteinhausen",
  email: "alexandra.konstantinopoulou@example-holdings.co.uk",
};

// liveIn marks the person away, adds records with long words and returns
// the paths of their pages and their lists.
export async function liveIn(base, post) {
  const { dir } = await (await fetch(base + "/api/describe")).json();
  const db = new DatabaseSync(join(dir, "data.db"));
  db.prepare("INSERT OR REPLACE INTO _meta (key, value) VALUES (?, ?)").run("since:owner", new Date(Date.now() - 3600e3).toISOString());
  db.close();
  const made = [
    ["note", { title: LONG.link, body: `Sent by ${LONG.email}: ${LONG.link}` }],
    ["task", { title: `Write to ${LONG.email}` }],
    ["person", { name: LONG.name, email: LONG.email, organisation: "Rindfleischetikettierungsüberwachungsgesetz GmbH" }],
    ["habit", { name: "Say Rindfleischetikettierungsüberwachungsgesetz" }],
  ];
  const paths = [];
  for (const [type, fields] of made) {
    const res = await post(`/api/${type}`, fields);
    if (res.status !== 201) throw new Error(`livedin: ${type} refused: ${res.status} ${(await res.text()).slice(0, 160)}`);
    paths.push(`/t/${type}`, `/t/${type}/${(await res.json()).id}`);
  }
  return paths;
}
