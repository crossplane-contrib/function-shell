#!/bin/bash
# nss-wrapper-entrypoint.sh
#
# This entrypoint script enables running the function as an arbitrary UID
# (common in OpenShift and restricted Kubernetes environments).
#
# When a container runs as an arbitrary UID that doesn't exist in /etc/passwd,
# many tools (aws cli, git, ssh, etc.) fail because they can't look up the
# current user. nss_wrapper intercepts these lookups and provides a fake
# passwd entry.
#
# Usage: Configure your DeploymentRuntimeConfig to use this entrypoint:
#
#   containers:
#     - name: package-runtime
#       command: ["/scripts/nss-wrapper-entrypoint.sh"]
#       args: ["/function"]
#

set -e

[ "$#" -eq 0 ] && { echo "usage: $0 <command> [args...]" >&2; exit 64; }

USER_ID=$(id -u)
GROUP_ID=$(id -g)

# Pick a writable HOME: the assigned UID may not own the home directory that
# /etc/passwd (or the image's ENV HOME) points at.
if [ ! -w "${HOME:-/}" ]; then
    for candidate in /tmp/home /tmp; do
        if mkdir -p "$candidate" 2>/dev/null && [ -w "$candidate" ]; then
            export HOME="$candidate"
            break
        fi
    done
    [ -w "$HOME" ] || echo "warning: no writable HOME found ($HOME)" >&2
fi

# Enable nss_wrapper only when the current UID has no passwd entry.
if ! getent passwd "$USER_ID" >/dev/null 2>&1 && [ -w "$HOME" ]; then
    export NSS_WRAPPER_PASSWD=/tmp/passwd
    export NSS_WRAPPER_GROUP=/tmp/group

    cat /etc/passwd > "$NSS_WRAPPER_PASSWD"
    echo "runner:x:${USER_ID}:${GROUP_ID}:Function Runner:${HOME}:/usr/sbin/nologin" >> "$NSS_WRAPPER_PASSWD"

    cat /etc/group > "$NSS_WRAPPER_GROUP"
    if ! grep -q "^[^:]*:[^:]*:${GROUP_ID}:" "$NSS_WRAPPER_GROUP"; then
        echo "runner:x:${GROUP_ID}:" >> "$NSS_WRAPPER_GROUP"
    fi

    export LD_PRELOAD=libnss_wrapper.so
fi

exec "$@"
