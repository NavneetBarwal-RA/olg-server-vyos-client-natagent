#!/usr/bin/env bash
set -euo pipefail

# Legacy compatibility wrapper.
#
# Phase 4 replaced the old action not_implemented behavior with
# placeholder trace success flow, so this script now delegates to:
#   tests/scripts/phase4-real-nats-action-smoke.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

exec ./tests/scripts/phase4-real-nats-action-smoke.sh "$@"
