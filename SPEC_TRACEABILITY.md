# SPEC v0.1 追踪矩阵

| SPEC 要求 | 实现位置 | 验证证据 | 状态 |
| --- | --- | --- | --- |
| CLI list/validate/run | `cmd/holdout/main.go` | `go run ./cmd/holdout validate --suite public-v0` | 已完成 |
| 结果离线校验 | `cmd/holdout/main.go`, `internal/model/validation.go`, `internal/catalog/catalog.go` | `validate-result --strict`、重复任务回归测试 | 已完成 |
| SUT 元数据完整性 | `internal/model/model.go`, `internal/model/validation.go`, `schema/result.v1.json` | dry-run schema + strict 校验 | 已完成 |
| 任务版本与材料摘要 | `internal/catalog/catalog.go`, `tasks/catalog.yaml` | catalog 校验 + digest 回归测试 | 已完成 |
| 12 条确定性 attacker/oracle | `tasks/T001`–`T012` | `tools/oracle_contract.py` | 已完成 |
| Adapter v0 生命周期 | `internal/sut/adapter.go` | adapter 状态/清理测试 | 已完成 |
| Docker/runc 与 gVisor 工厂 | `internal/sut/adapter.go` | Linux 环境下需真实 Docker/runsc | 契约完成，端到端待验收 |
| sink 按 run ID 隔离且脱敏 | `internal/sink/sink.go` | sink 单元测试、私网绑定测试、unsafe/未注册 run ID 回归测试 | 已完成 |
| Oracle 输入事件数组稳定 | `internal/sink/sink.go` | 空事件数组回归测试 | 已完成 |
| Linux runner 二进制可复现 | `Makefile`, `.github/workflows/holdout.yml` | GOOS=linux GOARCH=amd64 交叉构建 | 已完成 |
| Linux runner 前置诊断 | `runner/firecracker/check-prereqs.sh` | Linux 专用 runner 只读 preflight | 脚本合约完成，真实环境待验收 |
| Guest 私网连通配置 | `runner/firecracker/launch.sh`, `runner/firecracker/guest-init.sh` | shell 语法、IPv4/CIDR 参数校验、Linux VM 网络启动 | 脚本合约完成，真实环境待验收 |
| SUT 不获得隐式宿主机网关 | `internal/sut/adapter.go`, `internal/runner/runner.go` | Go 测试、contract、Linux bridge 端到端待验收 | 契约完成，端到端待验收 |
| 任务镜像来源可复现 | `internal/sut/adapter.go`, `cmd/holdout/main.go` | adapter 测试、dry-run contract | 已完成 |
| Linux 安全门与 Firecracker marker | `internal/runner/environment.go` | marker 权限、内容、symlink 测试 | 已完成 |
| Firecracker 外层启动 | `runner/firecracker/launch.sh`, `runner/LINUX_RUNNER.md` | Linux 专用 runner shellcheck + VM 启动 | 脚本合约完成，真实启动待验收 |
| guest marker 与 CLI 启动 | `runner/firecracker/guest-init.sh`, `runner/firecracker/holdout-guest.sh` | Linux guest 启动、marker 完整性和退出清理 | 脚本合约完成，真实启动待验收 |
| 项目化 rootfs 构建 | `runner/firecracker/build-rootfs.sh` | Linux root 权限下 hash 校验、ext4 挂载和镜像启动 | 脚本合约完成，Linux 构建待验收 |
| JSON Schema 2020-12 | `schema/result.v1.json`, `tools/validate_result_schema.py` | schema validator | 已完成 |
| 超时与清理错误区分 | `internal/runner/runner.go` | Docker adapter 在 Linux 上执行超时路径 | 契约完成，端到端待验收 |
| 执行失败保留 sink 证据 | `internal/runner/runner.go` | runner 失败路径契约检查、Linux 超时验收 | 契约完成，端到端待验收 |
| 原子结果归档 | `internal/runner/runner.go` | `internal/runner/write_test.go` | 已完成 |
| 任务文档完整字段 | `tasks/T001`–`T012`, `tools/expand_task_docs.py`, `tools/task_docs_contract.py` | task docs contract 的章节、sink、复现命令和危险任务授权检查 | 已完成 |
| 正确/错误配置 oracle 稳定区分 | `tools/oracle_contract.py` | 12 条空证据/命中证据试跑 | 已完成 |
| 两套 SUT 分数有区分度 | Linux Firecracker runner | Docker/runc 与 gVisor 各跑 12 条 | 未完成 |
| 干净机器复现并发布分数 | Linux runner + 归档结果 | 两台干净机器、`--strict`、schema 校验 | 未完成 |

## 当前边界

Windows 开发机只执行契约检查和 dry-run；dry-run 结果全部为 `skipped`，不
能作为正式榜单分数。L4/L5/G 族任务只能在项目控制的 Firecracker guest 内
运行，并且需要显式授权。
