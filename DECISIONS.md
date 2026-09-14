# Holdout 决策记录

## ADR-0001：采用 Go + Python 混合栈

状态：已确认。

Go 负责生命周期、并发、VM 和 adapter；Python 负责任务 attacker、oracle 与数据分析。两者通过命令行和 JSON Schema 解耦，贡献者不必理解整个 runner。

## ADR-0002：Apache-2.0

状态：已确认，发布前仍需法律复核具体 NOTICE 文案。

Apache-2.0 覆盖核心代码、任务和文档。攻击脚本的授权边界通过 NOTICE 和文档约束，不创建与 OSI 不兼容的自定义许可证。

## ADR-0003：外层 VM 是 L4/L5 的必要条件

状态：已确定。

Firecracker 用于 Linux 开发和 CI。runner 必须验证 Linux 和 `/run/holdout/outer-vm` marker；无外层 VM 时拒绝 G 族和 L4/L5 任务；macOS 支持另见 ADR-0004。

## ADR-0004：暂不支持 macOS

状态：已确认。

当前只支持 Linux 开发和 CI，macOS 的 Lima/QEMU 适配列入后续路线图，避免 Phase 1 引入平台兼容矩阵。
