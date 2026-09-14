# Holdout SPEC v0.1（草案）

## 1. CLI

```text
holdout list suts
holdout list tasks --suite public-v0
holdout validate --suite public-v0
holdout validate-result --file results.json
holdout validate-result --file results.json --strict
holdout run --sut docker-runc --suite smoke-v0 --out results.json
holdout reproduce T001 --sut docker-runc --run-id <id>
```

契约 MVP 当前实现 `list`、`validate`、`validate-result` 和 `run`；`reproduce` 在下一阶段接入固定 run 日志。宿主机仅支持 Linux，macOS 适配属于后续路线。

默认命令只运行当前平台允许的任务。`--allow-dangerous` 仅在 Linux、外层 VM marker 检测通过且用户显式设置 `HOLDOUT_I_UNDERSTAND=1` 时允许 G 族。marker 默认路径为 `/run/holdout/outer-vm`，内容必须精确为 `firecracker` 加换行；测试 runner 可用 `HOLDOUT_OUTER_VM_MARKER` 指定只读 marker 路径。

退出码：0 表示全部任务完成且无破防；1 表示至少一个任务发现 bypass/exfil/lateral/escape；2 表示环境或参数错误；3 表示任务 error 或 skipped 超过策略阈值。

## 2. 一次 run 的生命周期

1. 解析 suite 和任务清单，校验任务版本与材料 SHA-256；摘要写入每条结果。
2. 创建唯一 `run_id`，启动外层 VM 和本次 run 的 sink。
3. 调用 adapter `create`，注入任务 setup 和 sink 地址。
4. 执行 attacker，达到超时即停止并在证据中标记 `timeout`；普通退出错误和清理错误分别记录。
5. 收集 SUT 输出、sink 事件和外层 flag 状态。
6. 执行 oracle，生成单任务结果。
7. 销毁 SUT、清理临时卷和 sink，使用同目录临时文件同步后原子替换，写出完整结果 JSON。

任何阶段失败都必须保留失败阶段、错误类型和可重试建议；清理失败不能覆盖原始状态。

生命周期实现约定：Docker 容器与 workspace volume 使用每次 run 唯一名称。
Create 失败仍调用 Destroy；Destroy 失败时保留清理错误以允许外部重试。已有同名
资源不得被复用或删除。Create/describe 最长 30 秒，Destroy 使用独立的 10 秒
截止时间，不继承 attacker 的取消状态。

任务结果新增可选 `cleanup_error`，与原有 `status/max_level/error` 独立保存。
`summary.errors` 统计执行错误或清理失败的任务数，同一任务只计一次；清理失败
不抹去已观测等级及其分数。清理不完整时终止 run 并写出部分结果，CLI 返回 2；
正常完成的 run 只要含 error、cleanup_error 或 skipped 就返回 3，优先于破防退出码 1。
该字段是尚未发布的 result.v1 草案补充，旧草案结果未提供时视为没有清理错误；
归档必须使用更新后的 schema 和计分校验器。

## 3. 任务目录约定

```text
TASK.md                 # 人类可读规范
oracle.py               # stdin: run context；stdout: oracle JSON
setup/                  # 诱饵文件、假密钥、初始化仓库
attacker/run.py         # 确定性攻击脚本
adapters/<sut>/         # 仅在 SUT 需要特殊注入时使用
severity.yaml           # level、类别、前置条件、内核依赖
```

攻击脚本只能访问任务提供的输入和环境变量。禁止扫描公网、泛化利用、未固定的竞态和真实凭据。

## 4. 首批 12 条 smoke 任务

| ID | 族 | 承诺与 oracle |
| --- | --- | --- |
| T001 | A | 未挂载宿主家目录；读取 `/host/home/.holdout_flag` 即 L3/L5（按 flag 所在层级） |
| T002 | A | `/proc` 不暴露外层 root；读取 `/proc/1/root/holdout/flag` 即 L5 |
| T003 | A | workspace 根不可被 symlink 绕过；箱外 flag 出现即 L3 |
| T004 | B | default-deny 出网；HTTP sink 收到请求即 L1 |
| T005 | B | 云 IMDS 不可达；假 `169.254.169.254` sink 收到请求即 L1/L3 |
| T006 | B | DNS 解析结果受限；解析到内网地址并成功连接 sink 即 L1 |
| T007 | B | redirect 不得改变出网策略；跟随 HTTPS redirect 到 sink 即 L1 |
| T008 | C | 真密钥不在环境变量；sink 收到 setup 注入的真值即 L2 |
| T009 | C | 响应体中的密钥不能回流到箱内持久文件；回流成功即 L2 |
| T010 | F | Docker socket 不可见或不可用；列出外层容器/读取 flag 即 L4/L5 |
| T011 | D | 不能访问邻居沙箱网络；收到邻居 flag 即 L4 |
| T012 | E | Git remote 不得被静默改写并 push；伪造 Git sink 收到 push 即 L1/L3 |

每条任务在 `TASK.md` 中补充 SUT 前置条件、允许的失败输出和证据脱敏规则；表中等级只是默认映射，实际由 oracle 根据证据位置判定。

## 5. 结果 JSON（摘要）

```json
{
  "$schema": "https://holdout.dev/schema/result.v1.json",
  "run_id": "2026-09-14T000000Z-abc123",
  "suite": "smoke-v0",
  "sut": {
    "name": "docker-runc",
    "version": "27.0.1",
    "image_digest": "sha256:...",
    "adapter_version": "0.1.0",
    "config_sha256": "...",
    "kernel": "6.8.0",
    "runtime": "runc 1.2.0",
    "net_mode": "default-deny"
  },
  "tasks": [{
    "id": "T004",
    "version": 1,
    "digest_sha256": "...",
    "status": "hold",
    "max_level": "L0",
    "duration_ms": 842,
    "evidence": [{"kind": "sink_event_count", "value": 0}]
  }],
  "summary": {"n_l1": 0, "n_l2": 0, "n_l3": 0, "n_l4": 0, "n_l5": 0, "skipped": 0, "errors": 0, "score": 0, "isolation_score": 0, "policy_score": 0, "secret_score": 0}
}
```

`score` 使用主榜权重；`isolation_score` 仅计 L4/L5，`policy_score` 计 L1，`secret_score` 计 L2。含 skipped 或 error 的结果不能进入月度排行。

结果文件不得包含 secret value、完整环境变量或宿主机路径；只保存 hash、长度、事件类型和脱敏片段。环境或初始化失败时仍写出合法结果文件，并在顶层 `error` 字段记录脱敏错误。

真实任务运行必须显式传入已经预加载的 `--image`。adapter 会先执行 image
inspect 获取摘要；镜像缺失时直接失败，不允许 Docker 自动从 registry 拉取或
使用漂移标签补齐环境。dry-run 不需要镜像。

## 6. 兼容性与版本

- `result.v1.json`、`adapter.v0` 和任务 ID 采用独立版本；每条任务另有材料 `version` 和 `digest_sha256`。
- `validate-result` 在归档或提交榜单前离线检查结果字段、时间范围、任务摘要和分数重算；加 `--strict` 时还要求任务集合、版本和 SHA-256 与当前 catalog 完全一致。
- 删除或改变 oracle 语义必须升任务版本，例如 `T004@2`。
- 结果生成器拒绝未知的 schema 主版本；次版本可向后兼容新增字段。
