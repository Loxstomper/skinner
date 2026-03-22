#!/usr/bin/env bash
set -euo pipefail

# Pre-iteration hook: find the next work item from bd (beads).
# Outputs JSON per the Skinner pre-iteration contract:
#   {"prompt": "..."}  — override the iteration prompt
#   {"done": true}     — signal no work remains, stop the loop

MAX_ATTEMPTS=100

for i in $(seq 1 $MAX_ATTEMPTS); do
  ITEM=$(bd ready --sort priority --json 2>/dev/null | jq -r '.[0] // empty')

  if [ -z "$ITEM" ]; then
    echo '{"done": true}'
    exit 0
  fi

  ITEM_ID=$(echo "$ITEM" | jq -r '.id')
  ITEM_TYPE=$(echo "$ITEM" | jq -r '.type')

  # Claim the work item
  bd update "$ITEM_ID" --assignee clanker --claim --json >/dev/null 2>&1

  # If it's a feature with all subtasks already closed, close it and loop to next
  if [ "$ITEM_TYPE" = "feature" ]; then
    CHILDREN=$(bd children "$ITEM_ID" --json 2>/dev/null)
    ALL_CLOSED=$(echo "$CHILDREN" | jq 'length > 0 and all(.status == "closed")')
    if [ "$ALL_CLOSED" = "true" ]; then
      bd close "$ITEM_ID" --reason "All subtasks completed" --json >/dev/null 2>&1
      continue
    fi
  fi

  # Fetch full item details and the prompt template
  DETAILS=$(bd show "$ITEM_ID" --json 2>/dev/null)
  PROMPT_TEMPLATE=$(cat PROMPT_BUILD.md)

  # Combine item details + prompt template into a single prompt
  jq -n --arg id "$ITEM_ID" --arg details "$DETAILS" --arg template "$PROMPT_TEMPLATE" \
    '{"prompt": ("You have been assigned work item `" + $id + "`.\n\n<work-item>\n" + $details + "\n</work-item>\n\n" + $template)}'
  exit 0
done

# Safety cap reached — all items were completed features
echo '{"done": true}'
