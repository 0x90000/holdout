# 契约 MVP 状态

## 已完成

- Go CLI：`list suts`、`list tasks`、`validate`、`run`。
- 12 条 Phase 1 smoke 任务目录，每条包含任务说明、确定性 attacker、oracle 和 severity metadata。
- Docker/runc 基线 adapter 与 gVisor/runsc adapter 工厂，共享 `create/exec/put/get/net_mode/describe/destroy` 协议。
- 外层 HTTP sink，按 task run ID 隔离事件并只保存 payload hash、长度和路径。
- L0-L5 结果映射、主榜计分、Isolation/Policy/Secret 子分数、skipped/error 统计。
- `schema/result.v1.json`、结果失败时的脱敏部分结果、Go/Python 契约检查和 GitHub Actions contract job。
- 任务 catalog 现在锁定每条任务的版本和目录材料 SHA-256，并把摘要写入结果，避免脚本或 oracle 静默漂移。
- `validate-result` 离线检查结果 schema 关键字段、时间范围、任务摘要和分数重算，并已接入 contract CI。
- contract CI 增加 12 条 oracle 的空证据/正证据契约试跑，验证正确配置和故意命中路径的稳定映射。
- contract CI 增加 JSON Schema 2020-12 实例校验；本地可用 `make schema-check` 检查归档结果。
- 提供 Linux Firecracker launcher 与 guest init 合约：TAP、API 配置、清理 trap 和只读 marker 的创建位置均已固定；仍需在专用 Linux runner 用项目镜像完成真实启动验收。
- 提供 rootfs 构建脚本和 guest-side CLI 启动入口：rootfs 输入必须由操作者提供并通过 SHA-256 校验，脚本不会隐式下载发行版或覆盖已有镜像。
- sink 默认 loopback 绑定；外层 VM 通过显式私网地址配置让内层 SUT 访问，避免开发机意外暴露收件端点。
- Docker adapter 在创建失败路径主动调用清理，镜像摘要获取失败会让 run 失败而不会生成不可复现的榜单元数据。
- attacker 超时现在以独立 `timeout` 证据标记，和普通退出错误、清理错误分开保存。
- 结果增加 `cleanup_error`：清理失败不会覆盖已观察的状态，但会计入错误并使 CLI 返回非成功状态。
- 结果写出采用同步临时文件和原子替换，避免中断产生半个 JSON 归档。
- 12 个任务文档均已补齐前置条件、失败模式、授权证据和复现命令。
- `task_docs_contract.py` 已接入 contract/CI，自动检查每条任务的安全章节、sink 证据说明和复现命令。
- 任务文档契约还会要求危险任务明确记录 Firecracker、`HOLDOUT_I_UNDERSTAND` 和 `--allow-dangerous` 授权门。
- Adapter 生命周期现在区分“已创建”状态，避免在 create 早期失败时重复销毁或掩盖原始错误。
- Makefile 提供单一 `make contract` 入口，串联代码、任务、oracle、dry-run 和结果归档校验。
- Windows 开发机提供等价的 `tools/contract.ps1`，避免依赖未预装的 make。
- `contract.py` 统一使用参数数组和 fail-fast 退出码，契约结果写入忽略的 `bin/contract-results.json`。
- `SPEC_TRACEABILITY.md` 逐项记录规范要求、实现位置、验证命令和剩余 gate。
- Docker 配置哈希现在包含镜像摘要，标签漂移不会被误认为同一配置。
- `EVAL_FIRST.md` 记录了 launcher 的权限/副作用评估以及当前仅做静态验证、未在 Windows 主机启动 VM 的边界。
- README 复现命令顺序已覆盖 validate → run → validate-result；CI 同时执行 shellcheck、oracle 契约和 JSON Schema 校验。
- CI 和本地 lint 统一检查 `tasks`、`tools` 的 Python 语法与 Ruff 规则，开发依赖版本已锁定。
- `validate-result --strict` 现在校验结果与 catalog 的完整任务覆盖及版本/摘要一致性，防止不完整结果进入榜单。
- `validate-result --strict` 拒绝重复任务 ID，避免“数量正确但漏项”的结果绕过完整覆盖检查。
- 结果中的 SUT 版本、内核、运行时、架构和网络模式现在始终是非空字段；无法探测时明确写为 `unavailable`，并由 Go 校验与 JSON Schema 同时约束。
- Docker adapter 不再自动添加 `host.docker.internal:host-gateway`；真实运行必须显式设置 guest 私网 `HOLDOUT_SINK_BIND_ADDR`，避免隐式宿主机网关暴露。
- sink 所有入口统一校验安全 run ID 字符集，避免 redirect/query 参数改变 run 隔离边界。
- sink 现在由 coordinator 注册允许的 task run ID，未知但格式合法的 ID 也会被拒绝，防止跨任务证据污染。
- sink 无事件时返回空数组而非 JSON `null`，保持 oracle 输入协议稳定。
- CI 和 Makefile 提供固定的 Linux amd64 无 CGO 构建产物，供专用 Firecracker runner 使用。
- 提供只读 Firecracker runner 前置检查脚本，执行前验证 KVM、kernel/rootfs、bridge 和 host marker 状态。
- launcher/guest init 现在通过受限 kernel 参数配置 guest 私网 `eth0`，为 sink 和内层 Docker 提供确定的可达地址。
- launcher 会严格校验 guest IPv4/CIDR 参数范围，错误地址在 VM 启动前失败。
- Docker/gVisor 真实运行现在必须传入预加载镜像，缺少 `--image` 时拒绝隐式 registry 拉取，保证离线与摘要可复现。
- attacker 超时或退出错误时仍保留本次 run 的 sink 事件计数，错误结果不计入正式分数但可用于判断是否发生了部分外带。
- Linux + Firecracker marker 安全门：没有 `/run/holdout/outer-vm` 时，L4/L5/G 族只能 skipped。

## 当前验证证据

在 Windows 开发机完成：

- `go test ./...`
- `go vet ./...`
- 12 条任务 catalog 校验
- 所有 Python 任务语法检查
- schema JSON 解析检查
- 编译后二进制 dry-run，结果包含 12 个 skipped、0 个 error，退出码为 3
- 非 Linux 真实运行被拒绝，并写出合法的部分结果 JSON
- Firecracker launcher 在 Windows 上未启动；Linux 专用 runner 需执行 shellcheck 和真实 VM gate
- rootfs 构建脚本和 guest 启动脚本在 Windows 上仅完成静态检查；需要 Linux root、loop mount、项目镜像和 Docker/gVisor 才能做启动验收

## 尚未完成的 Phase 1 gate

需要 Linux runner 才能完成：

1. 自建 Firecracker 镜像、网络 namespace、`/run/holdout/outer-vm` marker 和清理 trap。
2. 外层 VM 内运行 Docker/runc 与 sink 的端到端 12 条任务。
3. 同一套任务在 gVisor/runsc 上复跑，并确认两个 SUT 分数有区分度。
4. 在干净 Linux 机器复现结果后，再发布 `smoke-v0` 分数。

这些工作不是 Windows dry-run 的替代品，完成后才满足路线图中“外层嵌套 + 两个锚点 SUT”的完整验收。
