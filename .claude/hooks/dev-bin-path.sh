#!/bin/sh
# Claude Code SessionStart hook: puts tools/dev-bin of the main worktree first
# on PATH for the session's commands, so that dkt runs the build of main (see
# tools/dev-bin/dkt).
[ -n "${CLAUDE_ENV_FILE:-}" ] || exit 0
root=$(git -C "${CLAUDE_PROJECT_DIR:-.}" worktree list --porcelain 2>/dev/null | sed -n '1s/^worktree //p')
[ -n "$root" ] || exit 0
dir=$(printf '%s' "$root/tools/dev-bin" | sed "s/'/'\\\\''/g")
printf "export PATH='%s':\"\$PATH\"\n" "$dir" >>"$CLAUDE_ENV_FILE"
