# Real VyOS Lab Smoke

This directory contains manual/on-demand lab validation for the real VyOS path.
It is release evidence, not normal PR CI.

Lab artifacts are collected locally for review. They are not uploaded
automatically by the manual GitHub self-hosted workflow.

## What This Proves

The configure lab smoke proves:

- configure is submitted through real NATS and JetStream KV
- the running agent handles the command in real configure mode
- rendered config is applied to a real VyOS VM/device
- the local agent state checkpoint contains the submitted UUID
- resubmitting the same UUID reports the already-in-sync path
- evidence artifacts are collected for PR or release review

## Smoke vs Lab

`tests/smoke` contains CI-friendly smoke scripts. They start local NATS and use
placeholder behavior, so they do not need a VyOS device.

`tests/lab` contains manual real-device scripts. They require a reachable VyOS
VM/device, credentials, real NATS, and an agent configured for real mode.

## Required Lab Topology

- A real or disposable VyOS VM/device reachable over SSH.
- A NATS server reachable by both the controller script and the agent.
- A running `vyos-nats-agent` configured for the same target and NATS server.
- `agent.configure.mode: real` for real configure validation.
- A known-safe desired config fixture from `tests/lab/configs`.

Use a disposable lab target. The fixtures are intended to be small WAN/WAN+LAN
smoke configs, but they still change VyOS configuration.

## Required Environment

```bash
export REAL_VYOS_LAB_ACK=I_UNDERSTAND
export NATS_URL=nats://<nats-host>:4222
export VYOS_TARGET=vyos
export VYOS_HOST=<vyos-host-or-ip>
export VYOS_USER=vyos
export STATE_PATH=/tmp/vyos-nats-agent/state.json
```

Use exactly one SSH auth method:

```bash
export VYOS_PASSWORD='<password>'
```

or:

```bash
export VYOS_SSH_KEY=/path/to/private/key
```

Optional:

```bash
export DESIRED_CONFIG_FILE=tests/lab/configs/desired-vyos-wan-only-config.json
export ARTIFACT_DIR=tests/lab/artifacts/manual-run
export CONFIG_UUID=cfg-lab-$(date +%s)
export RPC_ID=real-vyos-configure-$(date +%s)
export RESUBMIT_SAME_UUID=true
export EXPECTED_VYOS_MATCH=OLG_APPLY_SMOKE_TEST
```

If the script should start the agent process on the current runner, set both:

```bash
export AGENT_BINARY=/path/to/vyos-nats-agent
export AGENT_CONFIG_FILE=/path/to/config.yaml
```

Otherwise the script assumes the agent is already running.

## Run Configure Smoke

```bash
./tests/lab/real-vyos-configure-smoke.sh
```

The script writes evidence into `ARTIFACT_DIR`, including:

- `phase9-summary.md`
- `configure-status.jsonl`
- `configure-result.jsonl`
- `agent.log`
- `vyos-before.txt`
- `vyos-after.txt`
- `state.json`
- `commands-run.txt`
- `environment-summary.txt`

The script does not print passwords, tokens, or private keys.

## Action Trace Smoke

Real VyOS trace action is currently deferred. The agent has a placeholder trace
executor, which is already covered by unit and mocked integration tests. The lab
script refuses to claim real trace evidence until a real platform trace executor
exists:

```bash
./tests/lab/real-vyos-action-trace-smoke.sh
```

That command exits with status `2` and writes a deferral summary artifact.

## Collect Evidence

After a local lab run:

```bash
ARTIFACT_DIR=tests/lab/artifacts/manual-run ./tests/lab/collect-lab-evidence.sh
```

After a GitHub self-hosted lab workflow run, evidence is also collected locally
on the runner under `tests/lab/artifacts/github-actions` in the checked-out
workspace. The workflow prints the exact path at the end of the run.

Artifacts are not uploaded automatically. Before attaching the artifact
directory, or an archive of it, to a PR or release note, review and sanitize
the files and remove any lab-specific data that should not leave the lab.

## Secret Safety

- Do not enable shell tracing with `set -x`.
- Do not put passwords, tokens, or private keys in `commands-run.txt`.
- `environment-summary.txt` records whether secret variables are set, not their
  values.
- Review `agent.log` and VyOS output before sharing outside the lab.
- Review `vyos-before.txt`, `vyos-after.txt`, `state.json`,
  `commands-run.txt`, and `environment-summary.txt` before attaching artifacts
  to a PR or release note.

## Known Limitations

- Real trace action is not implemented yet.
- The configure script validates a configurable marker string in VyOS output;
  set `EXPECTED_VYOS_MATCH` if your fixture uses a different marker.
- Different lab topologies may require a different desired config fixture.
- `/tmp` state paths are acceptable for disposable lab runs but should be
  configured appropriately for production deployments.

## Rollback / Revert Notes

Before running the configure smoke, the script captures `vyos-before.txt`.
After the run, it captures `vyos-after.txt`.

To revert, use the lab's standard VyOS rollback process or submit a known-good
desired config through the same NATS/KV path. Do not run the lab smoke against a
production router.
