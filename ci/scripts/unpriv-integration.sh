#!/bin/bash
set -e

chmod +s /usr/bin/newuidmap
chmod +s /usr/bin/newgidmap

export VOLUME_DRIVER=overlay-xfs
export GROOTFS_TEST_UID=1000
export GROOTFS_TEST_GID=1000
export STORE=/var/vcap/data/grootfs/store/unprivileged

echo "I AM INTEGRATION: ${VOLUME_DRIVER} (${GROOTFS_TEST_UID}:${GROOTFS_TEST_GID})"

ginkgo -race -r integration
