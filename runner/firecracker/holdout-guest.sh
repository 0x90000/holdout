#!/bin/sh
set -eu

# Guest-side entry point. The outer init has already attested the VM before
# this script is reached; this script only assembles explicit CLI arguments.

die() {
    echo "holdout-guest: $*" >&2
    exit 2
}

marker=${HOLDOUT_OUTER_VM_MARKER:-/run/holdout/outer-vm}
[ -f "$marker" ] || die "outer VM marker is missing"
[ ! -w "$marker" ] || die "outer VM marker must be read-only"
printf 'firecracker\n' | cmp -s - "$marker" || die "outer VM marker is invalid"
[ -x /usr/local/bin/holdout ] || die "/usr/local/bin/holdout is not installed"

suite=${HOLDOUT_GUEST_SUITE:-smoke-v0}
sut=${HOLDOUT_GUEST_SUT:-docker-runc}
out=${HOLDOUT_GUEST_OUT:-/var/lib/holdout/results.json}
image=${HOLDOUT_GUEST_IMAGE:-}
root=${HOLDOUT_GUEST_ROOT:-/opt/holdout}

case "$suite" in (*[!A-Za-z0-9._-]*|'') die "invalid HOLDOUT_GUEST_SUITE" ;; esac
case "$sut" in (*[!A-Za-z0-9._-]*|'') die "invalid HOLDOUT_GUEST_SUT" ;; esac
case "$out" in (/*) ;; (*) die "HOLDOUT_GUEST_OUT must be absolute" ;; esac
[ -d "$root/tasks" ] || die "guest task catalog is missing"
cd "$root"

mkdir -p "$(dirname "$out")"
set -- run --sut "$sut" --suite "$suite" --out "$out"
if [ -n "$image" ]; then
    set -- "$@" --image "$image"
fi
if [ "${HOLDOUT_GUEST_ALLOW_DANGEROUS:-0}" = "1" ]; then
    [ "${HOLDOUT_I_UNDERSTAND:-}" = "1" ] || die "dangerous run requires HOLDOUT_I_UNDERSTAND=1"
    set -- "$@" --allow-dangerous
fi

exec /usr/local/bin/holdout "$@"
