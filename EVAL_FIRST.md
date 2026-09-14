# Eval-first 记录

## 生命周期修复验证

只读审计已确认：启动失败会清空容器标识、volume 删除失败不能重试、attacker
成功路径忽略清理错误。现有两条 adapter 测试未执行失败和清理流程，不能证明这些语义。
本次采用本地模拟 Docker 命令结果与内存 adapter 验证所有权、失败重试、取消后的清理
以及结果保留；不启动 Docker、Firecracker 或攻击任务，不需要 skill 或子 Agent。
模拟测试只证明控制流程，真实资源删除仍须 Linux 集成验收。统一契约入口使用 Python
subprocess 参数数组，只写仓库 bin 下的二进制与 dry-run 结果，并检查每个退出码。

## 本轮范围

本轮涉及一个外层 Firecracker launcher 和一个 guest init 脚本。它们是
高权限自动化能力，只有在专用 Linux runner、项目控制的 kernel/rootfs、
私有 bridge 和明确授权下才应执行。

## 是否需要额外 skill/Agent

不需要调用 Codex skill、子 Agent、插件或外部 SaaS。Go/Python 代码、shell
脚本和文档均在本仓库内完成；不引入额外进程内耦合。

## 输入、权限和副作用评估

- launcher 输入仅来自显式环境变量：Firecracker 二进制、kernel、rootfs、
  bridge、TAP 名称和资源大小。
- launcher 要求 root，并创建 TAP、启动 Firecracker、配置 guest 网络；退出
  trap 会终止 VM、删除 TAP 和临时 socket 目录。
- guest init 只在 guest 内创建 `/run/holdout/outer-vm`，内容固定为
  `firecracker\n`，并设置为无写权限文件。
- sink 默认 loopback；需要内层 SUT 访问时，操作者必须显式指定外层私网
  地址，禁止公网绑定。

## 最小验证与结论

已执行 Go 单元/静态检查、Python oracle 契约检查、结果 schema 校验、任务
catalog 摘要校验和 launcher 文本审查。当前 Windows 主机没有 Linux 网络
namespace、Firecracker 和专用 rootfs，因此没有启动 launcher，也没有运行
L4/L5/G 族任务。只有在专用 Linux runner 完成真实启动、清理和两套 SUT
端到端复现后，才能把结果用于榜单。

## Rootfs 与 guest 启动入口评估

本轮新增的 rootfs 构建脚本会以 root 挂载 loop-backed ext4 镜像并复制 CLI、任务
和 schema，guest 启动脚本会在 VM 内执行 `holdout run`。这是高权限且会产生磁盘
镜像副作用的能力；不调用 skill、子 Agent 或外部自动化。输入 rootfs 必须由操作
者提供并先通过 SHA-256 校验，脚本拒绝覆盖已有镜像，也不在运行时下载发行版。

在 Windows 上只完成 `bash -n`、参数和路径审查；没有执行 mount、TAP、Docker 或
Firecracker。Linux runner 仍需在独立机器上验证镜像启动、Docker/gVisor 连通性和
退出清理，之后才可进入 Phase 1 真实 gate。

## Container sink 连通性评估

审计发现 Docker adapter 原先通过 `host.docker.internal:host-gateway` 隐式暴露
宿主机网关。该映射超出 benchmark 所需权限，且会让默认隔离姿态失真。本轮移除
映射，并要求非 dry-run 显式提供非 loopback 的私网 `HOLDOUT_SINK_BIND_ADDR`；
没有调用 skill、子 Agent 或外部自动化。当前仅完成代码、测试和契约检查，真实
Docker bridge 到 guest sink 的连通性仍需 Linux runner 验收。

## Image 来源评估

审计发现 image 为空时 Docker adapter 会隐式拉取 `python:3.12-alpine`，这会把公网
依赖和可漂移标签带入正式结果。本轮要求真实运行显式传入预加载 image，缺失时直接
失败；dry-run 不创建 adapter，因此仍可离线检查编排。不调用 skill、子 Agent 或
外部自动化。

## Linux preflight 评估

新增的 `check-prereqs.sh` 只执行只读检查：Linux、root、KVM、工具、kernel/rootfs
文件、bridge 和 host marker。它不创建接口、不挂载镜像、不写 marker、不启动
Firecracker，因此可作为高风险 launcher 前的最小试运行；不调用 skill、子 Agent
或外部 SaaS。

## Guest 网络配置评估

launcher 原先只把 TAP 接入 bridge，没有向 guest 提供确定的 `eth0` 地址，无法
保证 sink 与内层 Docker 可达。本轮增加受限的 `holdout.guest_ip` 和
`holdout.guest_gateway` kernel 参数，由 guest init 在 VM 内配置私网接口。参数
只允许数字、点、斜杠并有专用默认网段；未启动 Firecracker，仅完成 shell 语法和
契约审查。

launcher 现在还会在宿主机侧验证 guest IPv4 地址和 CIDR 前缀范围，避免错误网络
参数进入 Firecracker kernel command line；未启动 VM。

## Attacker 失败证据评估

attacker 超时或非零退出时仍可能在退出前向 sink 发送事件。本轮让 runner 在
清理后读取并保存该 run 的事件计数，同时保持任务状态为 `error`，避免把不完整
执行当作正式分数；没有调用 skill、子 Agent 或外部自动化。

## Sink run ID 隔离评估

审计发现 sink 的 run ID 原先只拒绝路径分隔符，不能充分约束 redirect 和查询
参数。本轮将事件、redirect、response 和 runs 查询统一限制为生成器使用的
字母数字及 `-_.:` 字符，并增加一个 HTTP 回归测试；不调用 skill、子 Agent 或
外部自动化。

## Sink run 注册评估

仅校验 run ID 格式仍允许 attacker 猜测其他任务的 ID 并污染证据。本轮增加由
coordinator 显式注册的 allowlist，事件、redirect、response 和 runs 查询都必须
命中已注册 ID；增加未知 ID 回归测试。实现不引入 skill、子 Agent 或外部自动化。
