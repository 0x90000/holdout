# Outer Runner Contract

The MVP keeps outer VM orchestration intentionally small and explicit. A
Linux Firecracker image that is prepared by the operator must mount a
read-only marker at `/run/holdout/outer-vm` containing exactly:

```text
firecracker
```

The marker is an attestation from the outer runner, not a user-facing switch.
It must be a regular file with mode `0444` (or another mode with no write bits)
and its bytes must be exactly `firecracker\n`; the CLI rejects writable marker
paths.
`HOLDOUT_OUTER_VM_MARKER` may point to an equivalent read-only path in CI. The
Holdout CLI still checks that the host is Linux and requires
`HOLDOUT_I_UNDERSTAND=1` when dangerous tasks are selected.

The outer image is responsible for keeping `/holdout/flag`, the sink, and the
SUT bridge on the outer side of the boundary. It must not use a developer
workstation filesystem as the flag source. The supplied launcher configures the
VM; the guest init creates this marker only after control has entered the guest.

Until a project-controlled guest image is prepared, use the contract runner
only for the non-dangerous smoke path; do not manually create the marker on a
host.

## Firecracker launcher contract

`firecracker/launch.sh` is the Linux launcher for a project-controlled VM. It
requires root, an existing bridge, an immutable kernel, and a project-built
rootfs. It creates a private TAP device, configures the Firecracker API, and
cleans up the process and TAP device on exit. It never writes the marker on the
host. The guest rootfs must install `firecracker/guest-init.sh` as
`/sbin/holdout-init`; that init creates the read-only marker only after control
has entered the guest. TAP and bridge names are restricted to Linux interface
name characters and the guest MAC must be a six-octet address.

Example (on a dedicated Linux runner, with paths reviewed by the operator):

```sh
sudo env HOLDOUT_KERNEL=/srv/holdout/vmlinux \
  HOLDOUT_ROOTFS=/srv/holdout/rootfs.ext4 \
  HOLDOUT_BRIDGE=holdout-br0 \
  bash runner/firecracker/launch.sh
```

The launcher is intentionally separate from `holdout run`: the VM image must
start the guest-side runner and sink wiring before invoking the CLI. Do not
create `/run/holdout/outer-vm` manually on a workstation or shared host.

The bridge supplied through `HOLDOUT_BRIDGE` must itself be dedicated to the
outer VM and connected to a private network namespace by the operator. The
launcher refuses to create a public interface; it does not attempt to infer or
reconfigure an existing host network topology.

The launcher passes `HOLDOUT_GUEST_IP` (default `192.168.127.2/24`) and
`HOLDOUT_GUEST_GATEWAY` (default `192.168.127.1`) as kernel arguments. Guest init
configures `eth0` before starting the runner; override both values together when
the dedicated bridge uses another private subnet.

Before launching, run the read-only preflight on the dedicated runner:

```sh
sudo env HOLDOUT_KERNEL=/srv/holdout/vmlinux \
  HOLDOUT_ROOTFS=/srv/holdout/rootfs.ext4 \
  HOLDOUT_BRIDGE=holdout-br0 \
  bash runner/firecracker/check-prereqs.sh
```

The preflight checks Linux, KVM, required binaries, regular kernel/rootfs files,
the existing bridge, and the absence of a host-side marker. It does not create
interfaces, mount images, write markers, or start Firecracker.

The sink binds to loopback by default. In an outer VM, set
`HOLDOUT_SINK_BIND_ADDR` to the VM's private bridge address (for example
`192.168.127.1:0`) so the inner Docker bridge can reach it; never expose it on
an internet-facing interface.

## Building the guest image

The repository provides `firecracker/build-rootfs.sh` for a reproducible image
assembly step. It does not download a distribution or overwrite an existing
image. The operator supplies a pinned rootfs archive and its SHA-256, plus the
Linux `holdout` binary and checked-out task/schema directories:

```sh
sudo env \
  HOLDOUT_ROOTFS_SOURCE=/srv/holdout/alpine-rootfs.tar \
  HOLDOUT_ROOTFS_SHA256=<64-hex-digest> \
  HOLDOUT_ROOTFS_IMAGE=/srv/holdout/rootfs.ext4 \
  HOLDOUT_HOLDOUT_BIN=/srv/holdout/bin/holdout \
  HOLDOUT_TASKS_DIR=/srv/holdout/tasks \
  HOLDOUT_SCHEMA_DIR=/srv/holdout/schema \
  bash runner/firecracker/build-rootfs.sh
```

The archive must contain a working `/sbin/init`, `cmp`, a Docker daemon/runtime,
and Python 3.12 for the public tasks. The build adds `/sbin/holdout-init`, the
guest runner, the CLI, tasks, and schema. It intentionally leaves
`/holdout/flag` to the outer image/operator so the flag is not copied from a
developer workstation.

The guest init starts `/usr/local/bin/holdout-guest` when present. The guest
runner accepts `HOLDOUT_GUEST_SUT`, `HOLDOUT_GUEST_SUITE`,
`HOLDOUT_GUEST_IMAGE`, and `HOLDOUT_GUEST_OUT`; dangerous tasks additionally
require both `HOLDOUT_GUEST_ALLOW_DANGEROUS=1` and
`HOLDOUT_I_UNDERSTAND=1`. It verifies the read-only Firecracker marker before
invoking the CLI.
