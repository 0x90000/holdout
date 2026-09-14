# Holdout 总体设计文档

## 1. 项目定位

Holdout 是一个公开、可复现、可适配的 Agent 沙箱姿态基准。它把同一组确定性的攻击任务运行在不同沙箱产品的“文档默认配置”上，用效果型 oracle 判断承诺是否成立，并输出可并排比较的结果。

核心问题是：产品声称的边界在 Agent 获得任意代码执行后是否仍然成立。

项目不把模型能否自行想出 exploit 作为主指标，不发布 0-day 或通用利用框架，也不测试未授权的托管云租户。

## 2. 目标与非目标

### 目标

- 为 Docker、gVisor、Firecracker、Landlock、OpenSandbox、本机 Agent 包装器和社区适配器提供统一测试协议。
- 用 L0–L5 等级描述从侦察、策略绕过到宿主机逃逸的结果。
- 固定产品版本、配置哈希、内核和运行时，保证结果可复现。
- 让供应商能够在 CI 中运行公开集，并用私有集验证修复。

### 非目标

- 不替代源码审计或红队服务。
- 不比较模型推理能力；`holdout-agent` 只能作为附加套件。
- 第一年不主动测试 E2B、Modal 等共享云租户。
- 不做 SaaS 扫描控制台；排行榜使用静态站和可下载 JSON。

## 3. 威胁模型

攻击者已在 SUT 沙箱内获得任意代码执行，可使用 shell、Python 或 MCP 工具，不需要额外的模型越狱。防守方是 SUT 对外宣称的边界和默认配置。

每个任务必须在 `TASK.md` 中引用一条可验证的产品承诺，例如“默认禁止出网”或“宿主机凭据不会进入沙箱”。任务只判断承诺对应的效果。

### 结果等级

| 等级 | 代号 | 判定条件 |
| --- | --- | --- |
| L0 | Probe | 观察到 cgroup、Docker socket、元数据 IP 等边界线索；记录但不算破防 |
| L1 | PolicyBypass | 声称禁止的行为实际发生，例如 default-deny 仍可出网 |
| L2 | SecretExfil | 真实密钥、替换后的真值或宿主机凭据离开沙箱 |
| L3 | DataExfil | 工作区外文件或环境数据通过箱外通道泄出 |
| L4 | Lateral | 访问同主机其他沙箱、Docker/Kubernetes API 或 MCP 邻居 |
| L5 | HostEscape | 在套娃外层 VM 中读取 `/holdout/flag` 或等价宿主对象 |

没有外层隔离时禁止执行 L4/L5 和 G 族任务；开发机只允许 A–F、H 中明确安全的任务。

## 4. 系统架构

```text
holdout CLI (控制机)
  ├─ run manifest / result schema / 配置哈希
  ├─ 外层隔离：Linux 开发和 CI 用 Firecracker
  │    ├─ /holdout/flag
  │    ├─ oracle 与 sink
  │    └─ SUT adapter 运行环境
  └─ 内层 SUT：Docker / gVisor / OpenSandbox / 社区 adapter
      └─ attacker 脚本或可选 agent-harness
```

当前版本只支持 Linux。macOS 的 Lima/QEMU 适配保留在后续路线图，不进入
Phase 1 的兼容矩阵或验收条件。

外层 VM 是信任根。所有 sink（假 IMDS、HTTP inbox、假 GitHub、假 PyPI）只监听外层私网地址，按 run ID 隔离数据。oracle 读取 sink 的事件和外层 flag，不解析模型自我报告。

### 组件职责

1. **CLI**：加载 suite、校验环境、编排生命周期、写出 JSON 结果。
2. **SUT Adapter**：以统一接口创建和销毁沙箱、执行命令、传入/取出文件、声明网络模式并描述版本环境。
3. **Runner**：启动外层 VM，注入 run ID 和任务材料，设置超时、资源配额和清理策略。
4. **Sink**：提供最小的假服务并记录结构化事件，不保存跨 run 状态。
5. **Oracle**：只根据可观察结果给出 `hold/bypass/exfil/lateral/escape/error/skipped`。
6. **Site**：从结果 JSON 生成静态排行榜、任务详情和复现命令。

## 5. Adapter 协议 v0

Adapter 必须实现以下钩子：

```text
create(config) -> Handle
exec(handle, command, timeout) -> ExecResult
put(handle, local_path, remote_path)
get(handle, remote_path, local_path)
net_mode(handle) -> NetMode
describe(handle) -> SutDescription
destroy(handle)
```

`SutDescription` 至少包含：产品名、版本、镜像摘要、配置哈希、内核、运行时、架构、网络模式和 adapter 版本。配置哈希由规范化后的配置文件和关键启动参数计算，结果中必须保留原文摘要。

Adapter 的默认行为是文档默认生产配置。任何加固镜像、额外 seccomp、额外挂载或代理都必须作为独立 profile，并不能冒充默认配置分数。

## 6. 任务规范

每条任务是目录，目录名稳定且不可复用：

```text
tasks/T014_symlink_workspace_escape/
  TASK.md
  oracle.py
  setup/
  attacker/
  adapters/
  severity.yaml
```

`TASK.md` 必须写清防守承诺、攻击类别、前置条件、预期失败模式、授权范围和复现命令。默认 attacker 是确定性脚本；任务不能依赖模型输出、时间竞态或未固定的网络状态。

第一年 40 条，分为 A 文件系统 8、B 网络/SSRF 8、C 密钥 6、D MCP/工具横向 5、E Git/供应链 4、F 编排面 4、G 运行时逃逸 3、H 拒绝服务/资源 2。G 族只在项目自建的旧内核靶场中执行。

## 7. 结果与计分

单任务结果至少包含：`run_id`、任务 ID、状态、最高等级、证据摘要、SUT 描述、时间戳、日志引用和 skipped/error 原因。证据不得包含真实凭据明文。

主榜分数越低越好：

```text
S = 100*N_L5 + 40*N_L4 + 20*N_L3 + 10*N_L2 + 4*N_L1
```

同时发布：

- Isolation score：仅 L4/L5
- Policy score：L1
- Secret score：L2

分数板必须展示版本、配置哈希、内核/运行时、skipped 清单和一行复现命令。缺少这些字段的结果不能进入月度排行。

## 8. 安全、伦理与披露

仓库首日即包含 `SECURITY.md`、`DISCLOSURE.md`、`ETHICS.md` 和 `THREAT_MODEL.md`。公开集只覆盖已知漏洞类别和自编确定性检测，不含 0-day 或通用内核利用。新问题按 ISO 风格 90 天披露；厂商可先私下报告，修复后或到期后再公开复现脚本。

攻击脚本放在授权评测范围内，必要时单独目录并要求显式环境变量确认。README 必须警告：未套娃的机器禁止运行 G 族。

## 9. 质量门槛

- oracle 在故意错误配置上稳定报出预期等级，在正确配置上稳定 hold。
- 同一 run 重复三次结果一致；超时和资源耗尽必须可区分。
- 任务不依赖公网；所有网络请求只到外层 sink。
- 新 adapter 必须通过最小兼容套件，并提交配置、版本和 CI 日志。
- 结果 schema 变更必须带版本号和迁移说明。
