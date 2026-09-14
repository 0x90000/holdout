#!/usr/bin/env bash
set -Eeuo pipefail

# Read-only preflight for a dedicated Linux runner. This script does not create
# interfaces, mount images, write markers, or start Firecracker.

die() {
  echo "firecracker-prereq: $*" >&2
  exit 2
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

require_file() {
  [ -f "$1" ] || die "required regular file not found: $1"
}

[ "$(uname -s)" = "Linux" ] || die "Linux is required"
[ "$(id -u)" -eq 0 ] || die "run as root on a dedicated runner"
[ -c /dev/kvm ] || die "/dev/kvm is not available"

: "${HOLDOUT_KERNEL:?set HOLDOUT_KERNEL to the kernel image}"
: "${HOLDOUT_ROOTFS:?set HOLDOUT_ROOTFS to the rootfs image}"
: "${HOLDOUT_BRIDGE:=holdout-br0}"
: "${FIRECRACKER_BIN:=firecracker}"
require_file "$HOLDOUT_KERNEL"
require_file "$HOLDOUT_ROOTFS"
for command in curl ip mktemp "$FIRECRACKER_BIN"; do
  require_cmd "$command"
done
case "$HOLDOUT_BRIDGE" in
  (""|*[!A-Za-z0-9_.-]*) die "invalid bridge name: $HOLDOUT_BRIDGE" ;;
esac
[ "${#HOLDOUT_BRIDGE}" -le 15 ] || die "bridge name is longer than Linux IFNAMSIZ"
ip link show "$HOLDOUT_BRIDGE" >/dev/null 2>&1 || die "bridge does not exist: $HOLDOUT_BRIDGE"

if [ -e /run/holdout/outer-vm ] || [ -L /run/holdout/outer-vm ]; then
  die "refusing to run with a host-side outer-vm marker"
fi

echo "firecracker prerequisites passed (bridge=$HOLDOUT_BRIDGE)"
