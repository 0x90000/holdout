# Contributing to Holdout

Holdout is a benchmark for sandbox promises. Contributions should make a result
more reproducible, easier to audit, or more useful to adapter authors.

## Before opening a change

Read [AGENT.md](AGENT.md), [SPEC.md](SPEC.md), and [THREAT_MODEL.md](THREAT_MODEL.md).
They define the supported stack, result contract, task safety rules, and the
Linux-only Phase 1 boundary.

Keep changes focused. A new task should not require changes to the Go runner;
an adapter should expose the product's documented default configuration rather
than a private hardening profile.

## Adding a task

Create a new stable directory under `tasks/` containing:

```text
TASK.md
attacker/run.py
oracle.py
severity.yaml
```

The task document must state the product promise, preconditions, expected failure
mode, authorization and evidence rules, and a reproduction command. Attackers
must be deterministic, offline-capable, and limited to synthetic fixtures and
the local sink. Do not add real credentials, public endpoints, 0-days, or a
general exploit framework.

After changing task material, update its catalog version and SHA-256:

```bash
go run ./cmd/holdout validate --suite public-v0
```

## Adding an adapter

Implement the adapter contract in `internal/sut/adapter.go`: create, exec, put,
get, net mode, describe, and destroy. Report the product version, image digest,
runtime, kernel, architecture, network mode, and normalized configuration hash.
Do not add implicit mounts, privileges, host gateway mappings, registry pulls, or
network exceptions. Submit those choices as a named profile with separate
results.

## Checks before submission

Run the smallest relevant checks first, then the full contract:

```bash
go test ./...
go vet ./...
python -m compileall -q tasks tools
python -m ruff check tasks tools
python tools/oracle_contract.py
python tools/task_docs_contract.py
python tools/contract.py
```

The project deliberately avoids broad unit-test coverage. Add a test only when
it protects a necessary protocol, lifecycle, oracle, cleanup, or security path.

## Security reports

Do not open a public issue with a new vulnerability or sensitive evidence. Follow
[SECURITY.md](SECURITY.md) and [DISCLOSURE.md](DISCLOSURE.md) instead. Run L4/L5
or G-family tasks only on an authorized, dedicated Firecracker runner.

## Commit style

Use a short imperative subject with a clear scope, for example:

```text
feat: add OpenSandbox adapter
docs: clarify Linux runner setup
fix: reject duplicate result tasks
```

Keep generated binaries, result files, Python caches, and local runner secrets
out of commits.
