#!/bin/sh
# Opens a sameway workspace in your browser. Takes the same arguments as
# `sameway open`:
#   ./open-sameway.sh --workspace ~/work/my-workspace
#   ./open-sameway.sh --page /design
#
# Without --workspace it uses $SAMEWAY_WORKSPACE, or the folder you ran it from
# when that folder is a workspace. A fresh clone has neither, so it opens
# examples/workspaces/starter instead of stopping on an error. Make one of your
# own with `sameway init my-workspace`.
repo=$(dirname -- "$0")
pick=""
case " $* " in
*" --workspace"* | *" -workspace"*) ;;
*)
	if [ -n "$SAMEWAY_WORKSPACE" ]; then
		:
	elif [ -f "$PWD/workspace.yaml" ]; then
		pick=$PWD
	else
		echo "No workspace given, so this is the starter example."
		echo "For your own: sameway init my-workspace, then pass --workspace with it."
		echo
		pick=$(cd "$repo" && pwd)/examples/workspaces/starter
	fi
	;;
esac
cd "$repo" || exit 1
if [ -n "$pick" ]; then
	exec go run ./cmd/sameway open --workspace "$pick" "$@"
fi
exec go run ./cmd/sameway open "$@"
