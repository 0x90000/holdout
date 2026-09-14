# Holdout 路线图与验收标准

## Phase 0：立项冻结（第 1 周）

交付：仓库名和一句话定位、`THREAT_MODEL.md`、40 条任务标题清单、Adapter v0 接口、技术栈 ADR、Linux Firecracker 外层运行方案。macOS Lima/QEMU 只作为后续路线图，不作为本阶段交付。

验收：能在 5 分钟内说明 Holdout 与 SandboxEscapeBench、SandboxGym 的测量变量差异。

## Phase 1：一个产品跑通（第 2–6 周）

SUT 先做 Docker runc 默认配置，再做 gVisor 或 Nono。完成 12 条锚点任务：文件系统 3、网络 4、密钥 2、编排 1、邻居网络 1、Git 1。

交付：`holdout run --sut docker --suite smoke-v0 --out results.json`、外层嵌套 runner（见 `runner/firecracker/`）、sink、12 个 oracle、JSON Schema、GitHub Action、三页 `SPEC.md`。

生死线：第 6 周必须得到两个 SUT 明显不同的分数；否则先修架构和任务判定，不扩充任务数。

当前代码已完成本阶段的协议、任务和本地编排骨架；Firecracker 外层镜像、Linux 端到端执行和 Docker/gVisor 分数区分度仍是未完成的验收 gate，详见 `MVP_STATUS.md`。在这些 gate 完成前，发布物标记为“契约 MVP”，不发布正式分数。

## Phase 2：Adapter 生态与 public-v0.1（第 7–12 周）

适配 Docker、gVisor、OpenSandbox 以及 Nono 或 bubblewrap-claude。公开 20 条任务，以 A/B/C 为主，G 族最多 1 条且只在套娃 CI 执行。

交付：adapter 模板、贡献指南、静态分数板、英文技术文章、可下载结果和复现命令。邀请至少 5 个项目提交 adapter。

## Phase 3：形成惯例（第 4–8 月）

公开集扩至 40 条；建立 20 条私有 holdout（配置变体、TOCTOU、MCP 横向）；每月固定日期发布排行；提供 `holdout-agent` 附加套件和 Inspect-AI 导出。

验收：至少 4 个 adapter，其中 1 个来自第三方；至少 1 个真实产品依据 L1/L2 结果修复配置；CI 结果可由干净机器复现。

## Phase 4：标准层（第 9–18 月）

冻结 Adapter v1；私有集用于修复回归；向 OpenSandbox、Cube、Kubernetes SIG Agent Sandbox 提供 Holdout badge；发布任务分类和配置哈希影响的论文/报告。

商业化只允许私有集月跑和报告服务，保持核心基准开源。

## 90 天看板

| 周期 | 可验收结果 |
| --- | --- |
| 1–2 周 | Linux Firecracker 外层 + Docker + sink 完成一次 POST |
| 3–4 周 | 12 个 oracle 在故意错误/正确配置上分别稳定通过 |
| 5–6 周 | Schema、CI、三条复现命令、两台干净机器复现 |
| 7–8 周 | OpenSandbox 或 Nono adapter，补齐密钥任务 |
| 9–10 周 | 静态分数板、英文公告、发出 5 个 adapter 邀请 |
| 11–12 周 | 删除不稳定任务，发布 v0.1.0 |
