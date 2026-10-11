# Connecting: AI apps, agents and a model

Three things reach a workspace from outside it, and each is set up by the
owner alone: an **AI app** on this computer (Claude Desktop, Cursor, VS
Code and others) started over MCP, an **agent with a key** of its own
(`sameway agent add`), and the **model** the assistant thinks with. The
first two are on Your AI apps (`/apps`, from Help); the model is the
connect card on the chat when there is none
(`internal/server/connect`).

## Why it works this way

- **The person's real task is "let my app use my lists", not "edit an
  MCP file".** An app found on this computer is one press that writes its
  file, merged with every other server there, and then says the one thing
  to do in that app's own words. An app not found is a story of its own,
  closed until opened. One-click installs are what makes people use an
  integration at all
  ([VS Code, add MCP servers](https://code.visualstudio.com/docs/agent-customization/mcp-servers),
  [Cursor deeplinks](https://aiengineerguide.com/til/cursor-mcp-deeplink/)).
- **What a connected app may do is said once, before the first press, in
  plain words**: it can read and change everything, as you can, while it
  is open on this computer; what it changes is in Activity and can be
  undone; disconnect here at any time. A consent asks for the effect, not
  the technical name, in context, and says how to take it back before it
  is given
  ([Google, OAuth best practices](https://developers.google.com/identity/protocols/oauth2/resources/best-practices),
  [GDPR Art. 7(3)](https://noyb.eu/en/your-right-withdraw-your-consent-article-73)).
  It is said once above the apps, not on every row.
- **Exactly what a press starts is there to read.** Each app has a
  disclosure, "What Cursor starts", with the file written and the whole
  command, untruncated. The MCP security guidance asks this of any
  one-click setup of a local server, since the command runs with the
  person's own rights
  ([MCP security best practices, local server compromise](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices)).
  It stays closed: most people never open it, and a screen reader hears
  only its name until they do.
- **Taking it back is as easy as giving it.** A connected app has
  Disconnect beside it, one press, which takes Sameway out of its file and
  leaves the other servers as they were. No "Are you sure?": connecting
  again is one press, and a confirmation for what can be undone is clicked
  through unread
  ([NN/g, confirmation dialogs](https://www.nngroup.com/articles/confirmation-dialog/),
  [WCAG 3.3.4, reversible](https://www.w3.org/WAI/WCAG22/Understanding/error-prevention-legal-financial-data.html)).
  The page says honestly that an app already open lets go when it next
  starts.
- **An app pointed at another workspace is said so.** "Connected" means
  this workspace's entry, read from the file; one for another workspace is
  "Connected to another workspace, Work", and the press moves it here
  (Codex's file too, which used to refuse).
- **Agents with keys are listed where the apps are**, each with what it
  may do in plain words ("May look at everything, and change nothing"),
  when it was last used, and Take its key away, which logs the change with
  its Undo. People look for every app that reaches their account in one
  place, with when it was last used, and remove one there
  ([GitHub, reviewing your authorised integrations](https://docs.github.com/en/github/authenticating-to-github/keeping-your-account-and-data-secure/reviewing-your-authorized-integrations)).
  The section is not there when no agent has a key.
- **A model's key is checked before it is kept**, in a password field
  that can be pasted into, worded for the one company it is for, kept in
  the person's own settings and never the workspace
  ([NIST SP 800-63B, paste](https://pages.nist.gov/800-63-4/sp800-63b.html)).
- **Agents do the same.** Every press here is a page action an agent can
  post with `Accept: application/json`; taking a key away is the
  assistant's `take_agent_away` too.

## Not done, and why

- **Deep links** (`vscode:mcp/install`, `cursor://…/mcp/install`) need the
  app open and its own confirmation, and Claude Desktop, Windsurf and Codex
  have none. Writing the file is one way for all of them.
- **Making an agent's key on the page.** A key is shown once where it is
  made, and the assistant must never see one (`take_agent_away`'s
  description); the command line keeps it out of every page and
  conversation. Seeing and taking keys away is what a person needs here.
- **Narrower access for an app on this computer.** An app started over
  stdio is the owner, as the terminal is. Giving it less means a key of its
  own, which is what `sameway agent add --access view` is for.
- **A Show button on the key field, and forgetting a saved model key from
  a page.** The key is pasted, not typed, and is checked before it is
  kept, so a wrong one is said at once. A key is taken back at its
  company's keys page, which each story links.
