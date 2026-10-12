#!/bin/sh
# PreToolUse hook for Bash: Claude must never run migrations or change a database (see root CLAUDE.md).
# Best-effort denylist, not a guarantee (a read-only DB role would be). Fails closed if the command cannot be read.

block() {
	echo "Blocked: $1. Claude never runs migrations or changes a database; write the migration file and ask Akshay to run it." >&2
	exit 2
}

cmd=$(jq -er '.tool_input.command') || block 'could not read the Bash command'
flat=$(printf '%s' "$cmd" | tr '\n' ';')

# goose and make only count where a command starts, so the words inside commit messages or strings do not trigger.
start='(^|[;&|(`])[[:space:]]*([A-Za-z_][A-Za-z0-9_]*=[^[:space:]]*[[:space:]]+)*'
args='([[:space:]]+[^;&|]*)?[[:space:]]'
end='([[:space:]]|[;&|)]|$)'
goose="${start}(go[[:space:]]+tool[[:space:]]+)?([^[:space:];&|]*/)?goose${args}(up|up-by-one|up-to|down|down-to|redo|reset)${end}"
make="${start}make${args}migrate-(up|down)${end}"

# psql SQL often arrives through pipes, heredocs or quotes, so match the client and a write keyword anywhere.
psql='(^|[^a-z0-9_])(psql|pgcli)([^a-z0-9_]|$)'
sql='(^|[^a-z0-9_])(create|alter|drop|insert|update|delete|truncate|grant|revoke)([^a-z0-9_]|$)'

matches() {
	printf '%s' "$flat" | grep -Eiq "$1"
}

matches "$goose" && block 'state-changing goose command'
matches "$make" && block 'migration make target'
matches "$psql" && matches "$sql" && block 'psql with DDL or DML'
exit 0
