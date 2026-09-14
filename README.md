# Holdout

**A reproducible benchmark for the promises made by agent sandboxes.**

Holdout runs the same deterministic tasks against different sandbox products and
records what actually crossed the boundary. It measures a product's default
posture after arbitrary code execution, not whether a particular model happened
to invent an exploit.

[![CI](https://github.com/0x90000/holdout/actions/workflows/holdout.yml/badge.svg)](https://github.com/0x90000/holdout/actions/workflows/holdout.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20runner-lightgrey.svg)](TECH_STACK.md)

> **Current release: Contract MVP.** The CLI, task suite, adapters, result
> protocol, sink, and Firecracker runner contract are in the repository. Formal
> scores wait for a dedicated Linux runner to complete the end-to-end gate.

## What it measures

Each task corresponds to a concrete product promise: no host files, no unwanted
network access, no secret leakage, no Docker socket, no neighbor access. The
attacker is fixed and offline-capable. An oracle looks only at observable effects
such as a sink event or a synthetic flag crossing the boundary.

Results use six levels:

| Level | Meaning |
| --- | --- |
| L0 | Probe observed; no boundary violation |
| L1 | Policy bypass |
| L2 | Secret exfiltration |
| L3 | Data exfiltration |
| L4 | Lateral access to a neighbor or control plane |
| L5 | Host escape in the outer test VM |

The main score is intentionally simple and lower is better:

```text
100*L5 + 40*L4 + 20*L3 + 10*L2 + 4*L1
```

Isolation, policy, and secret sub-scores are included in every result.

## What is in the repository

- A Go CLI for catalog validation, task execution, and result validation.
- Docker Engine/runc and gVisor (`runsc`) adapter factories.
- Twelve `smoke-v0` tasks with deterministic Python attackers and oracles.
- A run-isolated HTTP sink that stores only event paths, lengths, and SHA-256 digests.
- `result.v1` JSON Schema, catalog digests, strict archive checks, and atomic result writes.
- A Linux Firecracker launcher, guest init, rootfs builder, and read-only preflight.
- Contract checks for Go, Python, shell, task documentation, and result archives.

## Quick start

The contract path works on a development machine without Docker or Firecracker.
It performs no attack run and does not produce a leaderboard score.

```bash
go test ./...
go run ./cmd/holdout validate --suite public-v0
go run ./cmd/holdout list tasks --suite public-v0
go run ./cmd/holdout run --sut docker-runc --suite smoke-v0 \
  --dry-run --out results.json
go run ./cmd/holdout validate-result --file results.json --strict
python tools/validate_result_schema.py results.json
```

Use `make contract` on Linux/macOS shells, or:

```powershell
powershell -ExecutionPolicy Bypass -File tools/contract.ps1
```

The contract runner expects Go, Python 3.12, and the locked development tools
in `requirements-dev.txt`.

## Running a real suite

Real execution is Linux-only. The image must already be loaded locally; Holdout
will not pull a floating registry tag. The sink must bind to a dedicated private
guest address so the container can reach it:

```bash
export HOLDOUT_SINK_BIND_ADDR=192.168.127.2:0
go run ./cmd/holdout run \
  --sut docker-runc \
  --suite smoke-v0 \
  --image holdout-task-image:local \
  --out results.json
```

For gVisor, use `--sut gvisor` with the same preloaded image and runner
configuration. The image digest, runtime, kernel, network mode, task versions,
and task material digests are written to the result.

L4/L5 and G-family tasks require all of the following:

1. A project-controlled Firecracker outer VM.
2. A read-only `/run/holdout/outer-vm` marker containing exactly `firecracker\n`.
3. `--allow-dangerous` and `HOLDOUT_I_UNDERSTAND=1`.

Do not create the marker on a workstation. See [runner/LINUX_RUNNER.md](runner/LINUX_RUNNER.md)
for the dedicated runner procedure.

## Task layout

```text
tasks/T001/
  TASK.md                 # promise, preconditions, evidence, reproduction
  attacker/run.py         # deterministic, offline-capable probe
  oracle.py               # effect-based result mapping
  severity.yaml           # level and safety metadata
```

Every task has a catalog version and material SHA-256. Changes to an attacker,
oracle, or task document must update the catalog version or create a new task ID.

## Design constraints

- Go owns the CLI, lifecycle, adapters, sink, and VM orchestration.
- Python 3.12 owns attackers, oracles, and offline task tooling.
- Linux plus Firecracker is the supported Phase 1 environment. macOS Lima/QEMU
  support is a later roadmap item.
- Public tasks use synthetic flags and secrets only. No 0-days, real credentials,
  public cloud tenants, or general exploit frameworks belong here.
- Tests cover necessary regression paths; the project does not chase unit-test
  volume for its own sake.

## Documentation

- [Design](DESIGN.md) and [SPEC v0.1](SPEC.md)
- [Roadmap](ROADMAP.md) and [MVP status](MVP_STATUS.md)
- [Threat model](THREAT_MODEL.md), [security](SECURITY.md), and [disclosure](DISCLOSURE.md)
- [Technology decisions](TECH_STACK.md) and [architecture decisions](DECISIONS.md)
- [SPEC traceability](SPEC_TRACEABILITY.md)
- [Firecracker runner](runner/README.md) and [Linux runner checklist](runner/LINUX_RUNNER.md)
- [Contributor constraints](AGENT.md)

## License

Code, task definitions, oracles, fixtures, and documentation are released under
[Apache-2.0](LICENSE). Third-party dependencies and images retain their own
licenses; see [NOTICE](NOTICE).
