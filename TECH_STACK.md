# 技术栈与开源协议决策稿

## 已确认基线（ADR-0001）

| 层 | 选择 | 原因 |
| --- | --- | --- |
| CLI、runner、adapter SDK、sink | Go 1.24+ | 单二进制、跨平台、并发和进程生命周期管理简单，Firecracker 有成熟 Go API |
| 任务 attacker 与 oracle | Python 3.12 | 便于编写确定性脚本、HTTP/Git/文件检查；通过 `uv` 锁定依赖 |
| 外层 Linux 隔离 | Firecracker | CI 可复现、边界清晰；镜像由项目构建 |
| macOS 开发 | 暂不支持 | 作为后续路线；当前只保证 Linux 开发与 CI，避免提前引入兼容矩阵 |
| SUT 容器 | Docker Engine/runc；可选 gVisor | 先做基线，再比较额外隔离层 |
| 协议与数据 | JSON + JSON Schema 2020-12 | 易被其他语言和 CI 消费，结果可长期归档 |
| 配置/构建 | Dockerfiles + Makefile/Taskfile | 降低贡献者门槛，后续再考虑 Nix |
| 测试 | Go test、Python 契约脚本、shellcheck、jsonschema | 覆盖编排、oracle、shell 和结果协议质量 |
| 文档/站点 | Markdown + MkDocs Material（Phase 2） | Phase 1 先用 Markdown，站点生成保持静态 |
| CI | GitHub Actions；Linux runner 执行套娃 smoke | 社区可见、日志可链接；重型 G 族可单独 self-hosted runner |

任务材料使用稳定的目录摘要：按规范化相对路径、文件长度和字节内容排序后计算 SHA-256，catalog 声明的版本与摘要会随每条结果写出。SUT 结果同时记录镜像摘要，且配置哈希覆盖镜像名、镜像摘要、网络模式和运行时。`.gitattributes` 固定文本任务材料使用 LF，避免跨平台换行转换造成漂移。

开发期 schema 工具依赖写入 `requirements-dev.txt` 并锁定版本；任务 attacker/oracle 继续保持 Python 标准库实现，运行任务不在线安装依赖。

发布与 CI 使用 Go 1.24；`go.mod` 的较低语言声明仅用于让早期离线开发机完成基础契约测试，不代表支持旧版 Go 的生产运行时。

## 关键取舍

- **为什么 CLI 用 Go**：Rust 的安全性有价值，但会让 adapter 生态和跨平台进程控制更难上手；Python 单二进制和 VM 生命周期不够稳。若未来需要更强的 sandbox 控制，可把 Firecracker 编排替换为独立 Rust runner，协议保持不变。
- **为什么任务用 Python**：攻击脚本属于测试材料，表达文件、网络和 Git 行为比性能更重要；每条任务使用锁定依赖和离线镜像，避免供应链漂移。
- **为什么不先上 Kubernetes**：Phase 1 需要可重复的单机嵌套环境；K8s adapter 在 Phase 2 以后作为外部 SUT 接入。

## 后续可讨论项

1. Phase 2 是否采用 MkDocs Material，或继续保持纯 Markdown。
2. 月榜是否允许 opt-in 上传结果到静态仓库；默认仍只写本地 JSON。
3. 私有 holdout 从 Phase 1 预留目录，还是到 Phase 3 再启用。

## 开源协议建议

- **核心代码、CLI、runner、adapter SDK、文档**：Apache-2.0。允许商业集成，带明确专利授权，适合吸引厂商提交 adapter。
- **任务定义、oracle、测试夹具**：Apache-2.0，与核心代码一致，便于供应商在 CI 镜像中复制。
- **攻击脚本**：默认 Apache-2.0；若律师要求增加授权边界，可附 `NOTICE-AUTHORIZED-EVALUATION.md`，但不把法律限制写进一个自定义“伪开源”许可证。
- **第三方依赖和镜像**：保留其原许可证与 SPDX 标识；发布物生成 SBOM。
- **私有 holdout**：不发布在仓库中，按合作协议授予有限评测许可；公开报告只披露任务 ID、等级和可复现证据摘要。

建议首日加入 `LICENSE`、`NOTICE`、`SECURITY.md`、`DISCLOSURE.md`、`ETHICS.md`，并在发布前让熟悉安全研究和开源许可的律师审阅攻击脚本授权文字。
