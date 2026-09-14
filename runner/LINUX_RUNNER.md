# Linux Runner 验收清单

这份清单用于专用、可重置的 Linux 机器。它不是共享开发机的安装指南；运行
Firecracker、Docker 和 G 族任务前，操作者必须确认机器、镜像和任务均已获授权。

## 固定环境

- Linux x86_64，`/dev/kvm` 可用，root 权限，独立磁盘和网络接口。
- 项目固定版本的 Firecracker、Docker Engine/runc、gVisor/runsc、Go 1.24 和
  Python 3.12。
- 已审查的 kernel、rootfs 归档和 SHA-256；rootfs 包含 Docker daemon/runtime、
  Python 3.12、`cmp`、`ip` 以及可用的 `/sbin/init`。
- 不配置公网路由。外层 bridge、guest 地址和 Docker bridge 使用专用 RFC1918
  网段，例如 bridge `192.168.127.1/24`、guest `192.168.127.2/24`。
- launcher 会把 guest 地址和 gateway 作为受限 kernel 参数传入，guest init 在
  启动 sink 前配置 `eth0`；换网段时同时设置 `HOLDOUT_GUEST_IP` 和
  `HOLDOUT_GUEST_GATEWAY`。

## 准备网络

下面的命令只在专用 runner 执行一次。bridge 不得连接到物理上行接口；guest rootfs
负责把 `eth0` 配置为 `192.168.127.2/24`，并把 `192.168.127.1` 作为需要时的
私网网关。若网络策略不需要出网，省略默认路由。

```sh
sudo ip link add holdout-br0 type bridge
sudo ip addr add 192.168.127.1/24 dev holdout-br0
sudo ip link set holdout-br0 up
```

确认 bridge 没有物理上行成员，再让 launcher 创建 TAP：

```sh
sudo env HOLDOUT_KERNEL=/srv/holdout/vmlinux \
  HOLDOUT_ROOTFS=/srv/holdout/rootfs.ext4 \
  HOLDOUT_BRIDGE=holdout-br0 \
  bash runner/firecracker/launch.sh
```

launcher 退出后应删除 TAP；bridge 由 operator 在验收结束时删除。不要在宿主机
创建 `/run/holdout/outer-vm`，该 marker 只能由 guest init 创建。

## Guest 与 SUT 验收

1. 在 guest 控制台确认 `/run/holdout/outer-vm` 是普通、无写权限文件，内容精确
   为 `firecracker\n`，`/holdout/flag` 来自外层镜像且不可写。
2. 确认 guest Docker daemon 已启动，`docker info` 成功；再确认
   `docker run --runtime runsc ...` 在安装 gVisor 时成功。
3. 设置 `HOLDOUT_SINK_BIND_ADDR=192.168.127.2:0`，让内层 Docker bridge 能访问
   guest sink；该地址只能是 guest 私网地址，不能是 `0.0.0.0` 或公网地址。
4. 先运行 `HOLDOUT_GUEST_SUT=docker-runc` 的 12 条任务，再用相同镜像和资源运行
   `HOLDOUT_GUEST_SUT=gvisor`。每套至少重复三次，记录版本、kernel、配置哈希和
   完整 stdout/stderr 摘要。
5. 只有在两套结果都通过 `validate-result --strict` 和 JSON Schema 校验，且没有
   `skipped`、`error` 或 `cleanup_error` 时，才可比较分数。分数有区分度后再在
   第二台干净 Linux 机器复现，最后才发布 `smoke-v0` 结果。

任何网络、Docker、Firecracker 或清理步骤失败，都保留诊断结果并停止正式榜单
流程；Windows dry-run 不能替代本清单。
