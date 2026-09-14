# Coordinated Disclosure

Holdout public tasks describe known vulnerability classes and deterministic
checks. They are not a vulnerability disclosure channel for a product.

When a run reveals a previously unknown product issue, stop testing that
variant, preserve only redacted evidence, and contact the vendor privately.
The default coordination window is 90 days. A vendor may request a practical
extension when a fix is actively being developed.

For benchmark-specific findings, maintainers may keep a private regression
task for up to 45 days and publish the reproducer after a fix or the agreed
deadline. Monthly rankings include only reproducible adapters, documented
configuration, and CI logs.
