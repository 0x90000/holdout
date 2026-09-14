# Holdout 公开任务 Backlog

任务 ID 一旦发布即不可复用；脚本、oracle 和严重级别可以在带版本后缀的新任务中修订。catalog 中的 `version` 与任务目录 `digest_sha256` 共同锁定可复现的任务材料。Phase 1 已实现的 12 条任务标为“已落地”，其余是 Phase 2/3 的规划标题。

| 族 | 数量 | 任务标题 | 状态 |
| --- | ---: | --- | --- |
| A 文件系统 | 8 | 宿主 home 挂载；`/proc` root 暴露；workspace symlink 绕过；TOCTOU 路径替换；`.ssh` 诱饵读取；hostPath 误挂载；设备节点探测；只读根文件系统写入 | 前 3 条已落地，其余规划 |
| B 网络/SSRF | 8 | 直连 IP 出网；metadata endpoint；内网 DNS；redirect 出网；IPv6 绕过；DNS rebinding；DoH 绕过；代理环境变量绕过 | 前 4 条已落地，其余规划 |
| C 密钥 | 6 | 环境密钥外带；响应密钥回流；文件密钥读取；Git credential 外带；日志密钥泄露；phantom 密钥真值混淆 | 前 2 条已落地，其余规划 |
| D MCP/工具横向 | 5 | 邻居 sandbox 网络；恶意 MCP 描述；跨 server 密钥读取；工具 token 重放；MCP 回调越权 | 第 1 条已落地，其余规划 |
| E Git/供应链 | 4 | Git remote push；依赖代理替换；Git hook 越权；构建缓存污染 | 第 1 条已落地，其余规划 |
| F 编排面 | 4 | Docker socket；特权容器；Kubernetes ServiceAccount 过权；宿主 API 发现 | 第 1 条已落地，其余规划 |
| G 运行时逃逸 | 3 | `/proc/1/root` 外层 flag；可写 cgroup 逃逸；固定旧 runc 配置错误 | 第 1 条已落地，其余规划 |
| H 拒绝服务/资源 | 2 | fork 炸弹影响邻居；磁盘填满影响邻居 | 规划 |

规划任务必须先通过 eval-first：验证任务是否必要、输入和副作用是否受控、oracle 是否能在正确/错误配置上稳定区分，再进入公开 suite。
