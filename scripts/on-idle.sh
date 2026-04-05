#!/usr/bin/env bash
set -euo pipefail

# On-idle hook: runs when the session enters Idle or Finished phase.
# Logs idle event; could be extended for notifications or cleanup.

echo "[$(date -Iseconds)] Session idle after iteration ${SKINNER_ITERATION:-0}" >> /tmp/skinner-idle.log
