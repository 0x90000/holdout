#!/bin/sh
set -eu

# Install this as /sbin/holdout-init in the project-controlled guest rootfs.
# It is the only component allowed to attest the outer VM to the Holdout CLI.

mkdir -p /run/holdout
umask 022
if [ -L /run/holdout ] || [ ! -d /run/holdout ]; then
    echo "holdout-init: /run/holdout is not a directory" >&2
    exit 1
fi

marker=/run/holdout/outer-vm
if [ -L "$marker" ]; then
    echo "holdout-init: marker must not be a symlink" >&2
    exit 1
fi
if [ -e "$marker" ]; then
    [ -f "$marker" ] || {
        echo "holdout-init: marker must be a regular file" >&2
        exit 1
    }
    printf 'firecracker\n' | cmp -s - "$marker" || {
        echo "holdout-init: existing marker has unexpected content" >&2
        exit 1
    }
else
    printf 'firecracker\n' > "$marker"
fi
chmod a-w,a+r "$marker"

# Configure the private guest interface from launcher-controlled kernel args.
# Values are restricted by launch.sh before they enter the Firecracker config.
guest_ip=
guest_gateway=
for arg in $(cat /proc/cmdline); do
    case "$arg" in
        holdout.guest_ip=*) guest_ip=${arg#holdout.guest_ip=} ;;
        holdout.guest_gateway=*) guest_gateway=${arg#holdout.guest_gateway=} ;;
    esac
done
if [ -n "$guest_ip" ]; then
    command -v ip >/dev/null 2>&1 || {
        echo "holdout-init: ip command is required for guest networking" >&2
        exit 1
    }
    ip link set eth0 up
    ip addr replace "$guest_ip" dev eth0
    if [ -n "$guest_gateway" ]; then
        ip route replace default via "$guest_gateway" dev eth0
    fi
fi

if [ -x /usr/local/bin/holdout-guest ]; then
    exec /usr/local/bin/holdout-guest
fi

exec /sbin/init
