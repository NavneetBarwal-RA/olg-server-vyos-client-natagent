#!/usr/bin/env bash
set -euo pipefail

# Manual config validation smoke check
#
# What this validates:
# 1. Config path resolution and YAML load
# 2. Defaults + explicit YAML override behavior
# 3. Config validation rules
# 4. Conversion to agentcore.Config
#
# Command executed:
# go run ./cmd/vyos-nats-agent \
#   --config ./config.example.yaml \
#   --validate-config \
#   --print-effective-config
#
# Expected success indicator:
# configuration valid

go run ./cmd/vyos-nats-agent \
  --config ./config.example.yaml \
  --validate-config \
  --print-effective-config
