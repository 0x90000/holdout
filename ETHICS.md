# Ethics and Authorized Use

Holdout measures whether a sandbox keeps documented promises after an Agent
already has code execution. It is intended for owned test environments,
vendor-provided self-hosted deployments, and explicitly authorized research.

- Use synthetic flags, keys, and sink endpoints supplied by the runner.
- Keep all network traffic inside the outer VM and its local sinks.
- Do not test shared hosted tenants, production hosts, or unpatched public
  kernels.
- Do not add 0-day exploits, generalized kernel exploit frameworks, or real
  credentials to the repository.
- Report configuration, version, kernel, skipped tasks, and reproducible
  logs when publishing results.

The project may reject a task or adapter that cannot provide a bounded,
reproducible, and authorized execution environment.
