#!/usr/bin/env bash
set -Eeuo pipefail

# Build an ext4 guest image from an operator-supplied, hash-verified rootfs
# archive. The archive is intentionally an input: distro selection and patch
# policy remain reviewable instead of being downloaded implicitly at runtime.

die() {
  echo "holdout-rootfs: $*" >&2
  exit 2
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

require_abs_file() {
  case "$1" in /*) ;; (*) die "path must be absolute: $1" ;; esac
  [ -f "$1" ] || die "regular file not found: $1"
}

require_abs_dir() {
  case "$1" in /*) ;; (*) die "path must be absolute: $1" ;; esac
  [ -d "$1" ] || die "directory not found: $1"
}

: "${HOLDOUT_ROOTFS_SOURCE:?set HOLDOUT_ROOTFS_SOURCE to a local rootfs archive}"
: "${HOLDOUT_ROOTFS_SHA256:?set HOLDOUT_ROOTFS_SHA256 to the archive SHA-256}"
: "${HOLDOUT_ROOTFS_IMAGE:?set HOLDOUT_ROOTFS_IMAGE to the output ext4 image}"
: "${HOLDOUT_HOLDOUT_BIN:?set HOLDOUT_HOLDOUT_BIN to the Linux holdout binary}"
: "${HOLDOUT_TASKS_DIR:?set HOLDOUT_TASKS_DIR to the checked-out tasks directory}"
: "${HOLDOUT_SCHEMA_DIR:?set HOLDOUT_SCHEMA_DIR to the checked-out schema directory}"

[ "$(id -u)" -eq 0 ] || die "must run as root on a dedicated Linux builder"
for command in sha256sum tar truncate mkfs.ext4 mount umount install grep awk cp; do
  require_cmd "$command"
done
require_abs_file "$HOLDOUT_ROOTFS_SOURCE"
require_abs_file "$HOLDOUT_HOLDOUT_BIN"
require_abs_dir "$HOLDOUT_TASKS_DIR"
require_abs_dir "$HOLDOUT_SCHEMA_DIR"
case "$HOLDOUT_ROOTFS_IMAGE" in /*) ;; (*) die "output image path must be absolute" ;; esac
if [ -e "$HOLDOUT_ROOTFS_IMAGE" ] || [ -L "$HOLDOUT_ROOTFS_IMAGE" ]; then
  die "refusing to overwrite existing image: $HOLDOUT_ROOTFS_IMAGE"
fi
mkdir -p "$(dirname "$HOLDOUT_ROOTFS_IMAGE")"
case "$HOLDOUT_ROOTFS_SHA256" in
  (????????????????????????????????????????????????????????????????) ;;
  (*) die "HOLDOUT_ROOTFS_SHA256 must contain 64 hexadecimal characters" ;;
esac
if ! printf '%s' "$HOLDOUT_ROOTFS_SHA256" | grep -Eq '^[0-9a-fA-F]{64}$'; then
  die "HOLDOUT_ROOTFS_SHA256 must contain 64 hexadecimal characters"
fi

actual_sha256="$(sha256sum "$HOLDOUT_ROOTFS_SOURCE" | awk '{print $1}')"
[ "${actual_sha256,,}" = "${HOLDOUT_ROOTFS_SHA256,,}" ] || die "rootfs archive SHA-256 mismatch"

: "${HOLDOUT_ROOTFS_SIZE_MIB:=1024}"
case "$HOLDOUT_ROOTFS_SIZE_MIB" in (*[!0-9]*|'') die "HOLDOUT_ROOTFS_SIZE_MIB must be a positive integer" ;; esac
[ "$HOLDOUT_ROOTFS_SIZE_MIB" -gt 0 ] || die "HOLDOUT_ROOTFS_SIZE_MIB must be positive"

workdir="$(mktemp -d /run/holdout-rootfs.XXXXXX)"
mounted=0
cleanup() {
  status=$?
  set +e
  if [ "$mounted" -eq 1 ]; then
    sync
    umount "$workdir/root"
  fi
  rm -rf -- "$workdir"
  if [ "$status" -eq 0 ]; then
    chmod 0644 "$HOLDOUT_ROOTFS_IMAGE"
  else
    rm -f -- "$HOLDOUT_ROOTFS_IMAGE"
  fi
  exit "$status"
}
trap cleanup EXIT INT TERM

mkdir -p "$workdir/root"
truncate -s "${HOLDOUT_ROOTFS_SIZE_MIB}M" "$HOLDOUT_ROOTFS_IMAGE"
mkfs.ext4 -F -q "$HOLDOUT_ROOTFS_IMAGE"
mount -o loop "$HOLDOUT_ROOTFS_IMAGE" "$workdir/root"
mounted=1
tar -xf "$HOLDOUT_ROOTFS_SOURCE" -C "$workdir/root" --no-same-owner --no-same-permissions

install -D -m 0755 "$HOLDOUT_HOLDOUT_BIN" "$workdir/root/usr/local/bin/holdout"
install -D -m 0755 "$(dirname "$0")/guest-init.sh" "$workdir/root/sbin/holdout-init"
install -D -m 0755 "$(dirname "$0")/holdout-guest.sh" "$workdir/root/usr/local/bin/holdout-guest"
mkdir -p "$workdir/root/opt/holdout/tasks" "$workdir/root/opt/holdout/schema" "$workdir/root/var/lib/holdout"
mkdir -p "$workdir/root/etc"
cp -a "$HOLDOUT_TASKS_DIR/." "$workdir/root/opt/holdout/tasks/"
cp -a "$HOLDOUT_SCHEMA_DIR/." "$workdir/root/opt/holdout/schema/"
printf 'source_sha256=%s\n' "$actual_sha256" > "$workdir/root/etc/holdout-rootfs-build"
chmod 0444 "$workdir/root/etc/holdout-rootfs-build"
sync
echo "built $HOLDOUT_ROOTFS_IMAGE (rootfs source sha256=$actual_sha256)"
