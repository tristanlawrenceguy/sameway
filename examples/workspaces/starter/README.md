# Sameway workspace

This folder is your whole setup. Commit it, share it, clone it on another
machine. `data.db` is the only file that stays local.

- `workspace.yaml` names the workspace and points the chat at a model.
- `schema/` holds one file per content type. `note.yaml` is an example you can
  change; `message.yaml` and `block.yaml` are used by the chat page.
- `components/` holds components of your own. Same folder layout as the
  built-in ones; a component here with a built-in name replaces it.
- `content/` holds exported records so they can live in git.

Run `sameway serve` in this folder and open the printed address.
