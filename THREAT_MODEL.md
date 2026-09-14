# Holdout Threat Model

## Scope

Holdout evaluates a sandbox after an attacker already has arbitrary code
execution inside the SUT. The attacker may run shell, Python, or an MCP tool;
model jailbreak ability is outside the primary benchmark. The defender is the
SUT's documented default boundary and configuration.

The benchmark is run only against systems and infrastructure owned by the
operator or explicitly authorized for testing. The public suite uses synthetic
secrets, local sinks, and fixed task inputs. It does not target shared cloud
tenants or publish general exploitation tooling.

## Trust Boundaries

The Firecracker outer VM is the trust root for L4, L5, and G-family tasks. It
contains the flag, sink, and SUT bridge. The runner accepts the outer boundary
only when it is on Linux and the read-only marker contains exactly
`firecracker` followed by a newline. A missing or invalid marker causes
dangerous tasks to be skipped.

The inner SUT is the system under test. Attacker code is untrusted and may
attempt only the behavior defined by its task. Oracles trust observable sink
events and outer-VM flag state, never the attacker's self-report.

## Impact Levels

| Level | Meaning | Observable evidence |
| --- | --- | --- |
| L0 | Probe | Boundary clue only; no prohibited effect |
| L1 | Policy bypass | A behavior documented as blocked succeeds |
| L2 | Secret exfiltration | A synthetic or authorized secret reaches an outer sink |
| L3 | Data exfiltration | Data outside the intended workspace reaches an outer sink |
| L4 | Lateral access | Another sandbox, orchestration API, or neighbor is reached |
| L5 | Host escape | The outer VM flag or equivalent host object is read |

The highest observed level is recorded per task. A task may report `hold`,
`bypass`, `exfil`, `lateral`, `escape`, `error`, or `skipped`.

## Safety Controls

- L4, L5, and G-family tasks require the Firecracker marker and explicit
  `--allow-dangerous` authorization with `HOLDOUT_I_UNDERSTAND=1`.
- Public attackers are deterministic, offline-capable, and limited to task
  inputs and runner-provided local endpoints.
- Sinks store event path, byte length, and payload SHA-256 only; result files
  contain no secret plaintext, full environment, or sensitive host paths.
- Every task has a timeout and the runner removes its container and workspace
  volume after execution.
- New vulnerability findings follow the coordinated process in
  `DISCLOSURE.md`; public tasks remain limited to known vulnerability classes.

## Out of Scope

Source-code auditing, model capability rankings, zero-day research, attacks on
unowned infrastructure, and production-kernel exploitation are outside this
benchmark. Runtime-escape exercises use only the project-controlled outer VM
images and documented fixtures.
