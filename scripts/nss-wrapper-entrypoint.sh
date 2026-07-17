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

USER_ID=$(id -u)
GROUP_ID=$(id -g)

# Only enable nss_wrapper if running as an arbitrary UID
# (not root and not the built-in nonroot user)
if [ "$USER_ID" != "0" ] && [ "$USER_ID" != "65532" ]; then
    export NSS_WRAPPER_PASSWD=/tmp/passwd
    export NSS_WRAPPER_GROUP=/tmp/group

    # Create a passwd entry for the current UID
    cat /etc/passwd > "$NSS_WRAPPER_PASSWD"
    echo "runner:x:${USER_ID}:${GROUP_ID}:Function Runner:/home/nonroot:/bin/bash" >> "$NSS_WRAPPER_PASSWD"

    # Create a group entry for the current GID if it doesn't exist
    cat /etc/group > "$NSS_WRAPPER_GROUP"
    if ! grep -q ":${GROUP_ID}:" "$NSS_WRAPPER_GROUP"; then
        echo "runner:x:${GROUP_ID}:" >> "$NSS_WRAPPER_GROUP"
    fi

    export LD_PRELOAD=libnss_wrapper.so
fi

exec "$@"
