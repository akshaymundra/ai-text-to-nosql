#!/bin/sh
# PreToolUse hook for Bash: Claude must never run migrations or change a database (see root CLAUDE.md).
cmd=$(jq -r '.tool_input.command // empty')

if printf '%s' "$cmd" | tr '\n' ' ' | grep -Eiq \
	-e 'goose.*[[:space:]](up|up-by-one|up-to|down|down-to|redo|reset)([[:space:]]|$)' \
	-e 'make.*[[:space:]]migrate-(up|down)' \
	-e '(psql|pgcli).*(^|[^a-z_])(create|alter|drop|insert|update|delete|truncate|grant|revoke)([^a-z_]|$)'; then
	echo 'Blocked: Claude never runs migrations or changes a database. Write the migration file and ask Akshay to run it.' >&2
	exit 2
fi
