#!/usr/bin/env bash
set -Eeuo pipefail

# Start one project-controlled Firecracker VM. This script deliberately does
# not create the outer-vm marker on the host: the guest init must create it
# only after the VM, network, and cleanup hooks are ready.

die() {
  echo "firecracker-launch: $*" >&2
  exit 2
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

require_file() {
  [ -f "$1" ] || die "required regular file not found: $1"
}

validate_path() {
  case "$1" in
    /*) ;;
    (*) die "path must be absolute: $1" ;;
  esac
  case "$1" in
    (*[!A-Za-z0-9_./-]*) die "path contains unsupported characters: $1" ;;
  esac
}

validate_json_value() {
  case "$1" in
    (*[!A-Za-z0-9._=,:/+\ -]*) die "value contains unsupported JSON characters: $1" ;;
  esac
}

validate_iface_name() {
  case "$1" in
    (""|*[!A-Za-z0-9_.-]*) die "interface name contains unsupported characters: $1" ;;
  esac
  [ "${#1}" -le 15 ] || die "interface name is longer than Linux IFNAMSIZ: $1"
}

validate_mac() {
  case "$1" in
    ([0-9A-Fa-f][0-9A-Fa-f]:[0-9A-Fa-f][0-9A-Fa-f]:[0-9A-Fa-f][0-9A-Fa-f]:[0-9A-Fa-f][0-9A-Fa-f]:[0-9A-Fa-f][0-9A-Fa-f]:[0-9A-Fa-f][0-9A-Fa-f]) ;;
    (*) die "HOLDOUT_GUEST_MAC must be a six-octet MAC address" ;;
  esac
}

validate_ipv4() {
  local value="$1" a b c d extra octet
  IFS=. read -r a b c d extra <<< "$value"
  [ -z "${extra:-}" ] || die "invalid IPv4 address: $value"
  for octet in "$a" "$b" "$c" "$d"; do
    case "$octet" in (""|*[!0-9]*) die "invalid IPv4 address: $value" ;; esac
    [ "$octet" -le 255 ] || die "invalid IPv4 address: $value"
  done
}

validate_ipv4_cidr() {
  local value="$1" address prefix
  case "$value" in
    (*/*) address=${value%/*}; prefix=${value#*/} ;;
    (*) die "guest IP must include a CIDR prefix: $value" ;;
  esac
  validate_ipv4 "$address"
  case "$prefix" in (""|*[!0-9]*) die "invalid IPv4 prefix: $value" ;; esac
  [ "$prefix" -le 32 ] || die "invalid IPv4 prefix: $value"
}

: "${FIRECRACKER_BIN:=firecracker}"
: "${HOLDOUT_KERNEL:?set HOLDOUT_KERNEL to an immutable Linux kernel image}"
: "${HOLDOUT_ROOTFS:?set HOLDOUT_ROOTFS to a project-controlled rootfs image}"
: "${HOLDOUT_TAP:=holdout-tap0}"
: "${HOLDOUT_BRIDGE:=holdout-br0}"
: "${HOLDOUT_GUEST_MAC:=06:00:AC:10:00:02}"
: "${HOLDOUT_VCPUS:=2}"
: "${HOLDOUT_MEM_MIB:=2048}"
: "${HOLDOUT_KERNEL_ARGS:=console=ttyS0 reboot=k panic=1 pci=off init=/sbin/holdout-init}"
: "${HOLDOUT_GUEST_IP:=192.168.127.2/24}"
: "${HOLDOUT_GUEST_GATEWAY:=192.168.127.1}"

[ "$(id -u)" -eq 0 ] || die "must run as root on a dedicated Linux runner"
require_cmd curl
require_cmd ip
require_cmd mktemp
require_file "$HOLDOUT_KERNEL"
require_file "$HOLDOUT_ROOTFS"
validate_path "$HOLDOUT_KERNEL"
validate_path "$HOLDOUT_ROOTFS"
validate_json_value "$HOLDOUT_KERNEL_ARGS"
case "$HOLDOUT_GUEST_IP" in
  (""|*[!0-9./]*) die "HOLDOUT_GUEST_IP must be an IPv4 address with prefix" ;;
esac
case "$HOLDOUT_GUEST_GATEWAY" in
  (""|*[!0-9.]*) die "HOLDOUT_GUEST_GATEWAY must be an IPv4 address" ;;
esac
validate_ipv4_cidr "$HOLDOUT_GUEST_IP"
validate_ipv4 "$HOLDOUT_GUEST_GATEWAY"
validate_iface_name "$HOLDOUT_TAP"
validate_iface_name "$HOLDOUT_BRIDGE"
validate_mac "$HOLDOUT_GUEST_MAC"
command -v "$FIRECRACKER_BIN" >/dev/null 2>&1 || die "firecracker binary not found: $FIRECRACKER_BIN"

case "$HOLDOUT_VCPUS:$HOLDOUT_MEM_MIB" in
  (*[!0-9:]*|:|*:*:) die "HOLDOUT_VCPUS and HOLDOUT_MEM_MIB must be positive integers" ;;
esac
[ "$HOLDOUT_VCPUS" -gt 0 ] || die "HOLDOUT_VCPUS must be positive"
[ "$HOLDOUT_MEM_MIB" -gt 0 ] || die "HOLDOUT_MEM_MIB must be positive"

workdir="$(mktemp -d /run/holdout-firecracker.XXXXXX)"
api_sock="$workdir/firecracker.sock"
fc_pid=""
tap_created=0

cleanup() {
  status=$?
  set +e
  if [ -n "$fc_pid" ] && kill -0 "$fc_pid" 2>/dev/null; then
    kill "$fc_pid"
    wait "$fc_pid" 2>/dev/null
  fi
  if [ "$tap_created" -eq 1 ]; then
    ip link delete "$HOLDOUT_TAP" 2>/dev/null
  fi
  rm -rf -- "$workdir"
  exit "$status"
}
trap cleanup EXIT INT TERM

ip link show "$HOLDOUT_BRIDGE" >/dev/null 2>&1 || die "bridge does not exist: $HOLDOUT_BRIDGE"
if ip link show "$HOLDOUT_TAP" >/dev/null 2>&1; then
  die "tap already exists: $HOLDOUT_TAP"
fi
ip tuntap add dev "$HOLDOUT_TAP" mode tap user "$(id -un)"
tap_created=1
ip link set "$HOLDOUT_TAP" master "$HOLDOUT_BRIDGE"
ip link set "$HOLDOUT_TAP" up

"$FIRECRACKER_BIN" --api-sock "$api_sock" >"$workdir/firecracker.log" 2>&1 &
fc_pid=$!
for _ in $(seq 1 100); do
  [ -S "$api_sock" ] && break
  kill -0 "$fc_pid" 2>/dev/null || die "firecracker exited during startup"
  sleep 0.05
done
[ -S "$api_sock" ] || die "firecracker API socket did not appear"

api() {
  curl --fail --silent --show-error --unix-socket "$api_sock" \
    -H 'Content-Type: application/json' "$@"
}

api -X PUT http://localhost/machine-config \
  -d "{\"vcpu_count\":$HOLDOUT_VCPUS,\"mem_size_mib\":$HOLDOUT_MEM_MIB,\"smt\":false}"
boot_args="$HOLDOUT_KERNEL_ARGS holdout.guest_ip=$HOLDOUT_GUEST_IP holdout.guest_gateway=$HOLDOUT_GUEST_GATEWAY"
validate_json_value "$boot_args"
api -X PUT http://localhost/boot-source \
  -d "{\"kernel_image_path\":\"$HOLDOUT_KERNEL\",\"boot_args\":\"$boot_args\"}"
api -X PUT http://localhost/drives/rootfs \
  -d "{\"drive_id\":\"rootfs\",\"path_on_host\":\"$HOLDOUT_ROOTFS\",\"is_root_device\":true,\"is_read_only\":false}"
api -X PUT http://localhost/network-interfaces/eth0 \
  -d "{\"iface_id\":\"eth0\",\"host_dev_name\":\"$HOLDOUT_TAP\",\"guest_mac\":\"$HOLDOUT_GUEST_MAC\"}"
api -X PUT http://localhost/actions \
  -d '{"action_type":"InstanceStart"}'

echo "firecracker VM started (pid=$fc_pid, tap=$HOLDOUT_TAP)"
echo "guest init must create a read-only /run/holdout/outer-vm containing firecracker\\n"
wait "$fc_pid"
