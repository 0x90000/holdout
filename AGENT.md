# Holdout Agent 工作约束

本文件是仓库级开发约束。所有参与 Holdout 的 Codex、自动化 Agent 和贡献者都必须遵守；更具体的目录约定可以在子目录追加 `AGENT.md`，但不得放宽本文件的安全边界。

## 已确定的技术栈

- **Go 是平台工程的主语言**：CLI、runner、SUT adapter SDK、生命周期管理、外层 VM 编排、sink 服务使用 Go。
- 发布与 CI 固定 Go 1.24；不要因本地旧版工具链而引入新的旧 API 兼容层。
- **Python 是任务工程的主语言**：attacker 脚本、oracle、任务辅助工具和离线分析使用 Python 3.12。
- Go 与 Python 通过命令行、环境变量和 JSON Schema 交互；不得引入隐藏的进程内耦合。
- Python 依赖必须锁定版本，优先使用 `uv`；任务运行不能依赖公网安装依赖。
- Linux 是当前唯一受支持的宿主开发/CI 平台。外层隔离使用 Firecracker；macOS 的 Lima/QEMU 支持属于未来路线，当前不得为它增加兼容层或 CI 矩阵。
- 外层 VM 是 L4/L5 和 G 族任务的必要条件。runner 必须同时检测 Linux 和由外层编排器注入的 `/run/holdout/outer-vm` marker（内容为 `firecracker`）；不能把任意环境变量当作隔离证明。
- sink 默认只绑定 loopback；当前 Docker/gVisor bridge runner 如需让内层 SUT 访问，必须显式设置 `HOLDOUT_SINK_BIND_ADDR` 为 guest 私网地址，禁止绑定公网接口或依赖 `host.docker.internal` 网关映射。
- Firecracker launcher 必须在专用 Linux runner 运行，使用项目控制的 kernel/rootfs、独立 TAP/bridge 和退出清理 trap；禁止在宿主机手工创建 marker，marker 只能由 guest init 以无写权限文件创建。
- guest rootfs 必须由 `runner/firecracker/build-rootfs.sh` 从操作者提供的本地归档构建；归档 SHA-256 必须显式校验，构建脚本不得运行时下载发行版或覆盖已有镜像。guest CLI 入口固定在 `/opt/holdout` 工作目录，避免从任意目录误发现任务集。
- SUT 首批支持 Docker Engine/runc 和 gVisor；OpenSandbox、Nono 等通过 adapter 接入，不改变核心协议。
- 结果协议使用 JSON Schema 2020-12；schema 主版本、adapter 主版本和任务 ID 版本彼此独立。
- 任务 catalog 必须为每条任务声明正整数 `version` 和目录材料 `sha256`；runner 校验后将两者写入每条结果，修改任务材料必须更新版本或生成新任务 ID。
- 归档或上榜前必须运行 `holdout validate-result`；结果摘要必须能由任务结果重新计算得到。
- 提交榜单必须额外使用 `holdout validate-result --strict`，确认结果完整覆盖 suite 且任务版本/摘要与 catalog 一致；环境失败的部分结果只能用于诊断。
- 归档结果还必须用 `tools/validate_result_schema.py` 按 `schema/result.v1.json` 校验；CI 可联网安装锁定主版本的 jsonschema，任务运行本身不得联网安装依赖。
- 构建和本地任务使用 Dockerfile + Makefile 或 Taskfile。除非有明确收益，不引入 Nix、Kubernetes 控制面或 SaaS 依赖。
- CI 使用 GitHub Actions；重型 G 族任务只能在隔离的专用 runner 上运行。
- Phase 1 使用 Markdown 作为文档入口；Phase 2 再评估 MkDocs Material，禁止提前建设复杂前端或 SaaS 控制台。

## 许可证与发布

- 核心代码、CLI、runner、adapter SDK、sink、任务定义、oracle、测试夹具和文档统一采用 Apache-2.0。
- 保留第三方依赖和镜像的原始许可证与 SPDX 标识，并为发布物生成 SBOM。
- 不创建带用途限制的“伪开源”许可证。授权评测、安全边界和披露流程写入 `SECURITY.md`、`DISCLOSURE.md`、`ETHICS.md` 与 `NOTICE`。
- 公开任务只包含已知漏洞类别和确定性检测脚本；不得提交 0-day、通用利用框架、真实凭据或未授权云租户测试代码。

## 任务与 oracle 约束

- oracle 只判断可观察效果（外层 flag、sink 事件、邻居对象），不得解析模型自我汇报。
- 默认 attacker 必须确定性、可离线、可复现；禁止依赖未固定的时间竞态、公网服务或模型版本。
- 真实任务运行必须使用操作者预加载且已记录摘要的镜像；adapter 不得因缺少 `--image` 自动从公网 registry 拉取标签镜像。
- 证据必须脱敏，结果 JSON 不得写入真实 secret、完整环境变量或宿主机敏感路径。
- 任务和 runner 解耦；新增任务不得修改 CLI 核心流程。

## 测试策略

- **禁止过度单元测试**。只为必要路径编写测试：结果 schema/计分、危险任务拒绝、adapter 生命周期、oracle 的关键判定和清理失败语义。
- 不为简单 getter、薄包装、静态数据或逐行复制实现编写测试；不要写与实现细节一一对应的“镜像测试”。
- 优先少量端到端 smoke 测试验证真实边界：外层 VM + sink + Docker/gVisor + 代表性任务。
- 每个新增测试必须说明它捕获的回归风险；如果集成测试已经覆盖该风险，不再增加重复单元测试。
- 运行必要检查即可：Go test、pytest、ruff、shellcheck 和 schema 校验。不要为了追求覆盖率引入无价值测试或降低交付速度。
- 本地优先运行 `make contract`；该入口不启动 Firecracker 或高风险任务，只验证契约和 dry-run。
- Windows 无 make 时运行 `powershell -ExecutionPolicy Bypass -File tools/contract.ps1`。

## Skill 与 Agent：eval-first

涉及任何 skill、子 Agent、插件或外部自动化能力时，必须先执行 **eval-first**：

1. 明确当前任务是否真的需要该能力，以及不用它的成本。
2. 检查能力的输入、输出、权限、网络和副作用，确认符合本文件的安全边界。
3. 选择最小的可验证试运行或只读步骤，先验证结果质量，再扩大范围。
4. 在工作记录或 PR 描述中写明采用/不采用的理由和验证结果。

未完成 eval-first 前，不得调用 skill、生成攻击性任务、启动子 Agent、扩大文件写入范围或运行高风险沙箱任务。对 G 族、L4/L5 和任何可能触及宿主机的操作，eval-first 还必须确认外层 VM 已建立且任务授权明确。

## 变更纪律

- 先更新设计/决策文档，再实现跨组件协议变更。
- 配置哈希、产品版本、内核和运行时必须进入结果；不得用自定义加固配置冒充默认配置分数。
- 保持变更小而可审查；不因“未来可能需要”提前加入抽象层、平台适配或测试矩阵。
- 提交前运行与改动直接相关的最小验证，并在说明中记录已验证项和已知限制。
