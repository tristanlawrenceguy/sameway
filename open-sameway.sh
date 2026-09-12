#!/bin/sh
# Opens this workspace in a browser. Run it with no arguments to use the
# nearest workspace.yaml or $SAMEWAY_WORKSPACE, or pass --workspace <dir>.
# If you have not made a workspace yet: sameway init my-workspace
cd "$(dirname "$0")" || exit 1
exec go run ./cmd/sameway open "$@"
