# Security Policy

Holdout is a security evaluation tool. Use it only against a SUT and an
environment that you own or are explicitly authorized to test.

Do not run G-family, L4, or L5 tasks on a developer workstation or on a
shared host. The runner requires an isolated outer VM and marker for those tasks. Public
tasks use synthetic secrets and local sinks; never replace them with real
credentials or public cloud endpoints.

To report a vulnerability in Holdout itself, open a private security report
with the affected commit, reproduction steps, impact, and whether a public
task or runner boundary is involved. Do not include real credentials or
unpatched exploit code in an issue.

We aim to acknowledge reports within 7 days, provide an initial assessment
within 14 days, and coordinate disclosure within 90 days. The timeline can be
shortened when a report is already being exploited or lengthened when a
coordinated fix needs more time.
