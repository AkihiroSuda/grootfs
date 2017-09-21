#!/bin/bash
set -e

chmod +s /usr/bin/newuidmap
chmod +s /usr/bin/newgidmap

echo "I AM INTEGRATION: ${VOLUME_DRIVER} (${GROOTFS_TEST_UID}:${GROOTFS_TEST_GID})"

args=$@
[ "$args" == "" ] && args="-r integration"
ginkgo -race $args
