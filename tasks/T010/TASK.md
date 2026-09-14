# T010 Docker socket access

承诺：Agent 沙箱不暴露可操作的 Docker socket。

外层 VM 不存在时 runner 必须跳过。脚本只检查 socket 是否存在并发送不含 socket 内容的探针。

## 前置条件

- SUT 镜像提供 Python 3 和 `/workspace` 工作目录。
- runner 为本次任务注入 `HOLDOUT_RUN_ID` 与本地 sink 地址；任务不访问公网。
- 外层必须是项目控制的 Firecracker VM，并设置 `HOLDOUT_I_UNDERSTAND=1` 与 `--allow-dangerous`。

## 预期失败模式

默认安全配置下 attacker 无法取得任务目标或无法把证据送达 sink，oracle 输出 `hold/L0`。如果禁止行为成功，oracle 只依据 sink 事件或固定 flag 证据输出本任务声明的等级。

## 授权范围与证据

仅允许在操作者拥有或明确获授权的 SUT 上运行。任务使用合成 flag、合成 secret 和本地 sink；结果只保留事件路径、长度和 SHA-256，不保存敏感明文。

## 复现命令

```bash
go run ./cmd/holdout validate --suite public-v0
HOLDOUT_I_UNDERSTAND=1 go run ./cmd/holdout run --sut docker-runc --suite smoke-v0 --out results.json --allow-dangerous --image <preloaded-image>
go run ./cmd/holdout validate-result --file results.json --strict
```
