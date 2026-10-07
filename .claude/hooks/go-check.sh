#!/bin/sh
# Stop hook: block once with the failure output so Claude fixes it; skip on the retry to avoid a loop.
if cat | grep -Eq '"stop_hook_active" *: *true'; then
	exit 0
fi

cd "$CLAUDE_PROJECT_DIR/server" || exit 0

if [ -z "$(git status --porcelain -- '*.go' go.mod go.sum)" ]; then
	exit 0
fi

if ! out=$(go vet ./... 2>&1 && go test ./... 2>&1); then
	printf 'go vet / go test failed:\n%s\n' "$out" | grep -v '\[no test files\]' | tail -n 60 >&2
	exit 2
fi
