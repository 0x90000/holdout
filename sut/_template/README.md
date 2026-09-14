# SUT Adapter Template

An adapter implements the interface described in `DESIGN.md` and
`internal/sut/adapter.go`:

`create`, `exec`, `put`, `get`, `net_mode`, `describe`, and `destroy`.

The default profile must match the product's documented production defaults.
Every result must include the product version, adapter version, normalized
configuration SHA-256, kernel, runtime, architecture, and network mode.

Adapters must not silently add mounts, privileges, network exceptions, or
custom hardening. Submit those as a named profile with separate results.
