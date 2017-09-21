#!/bin/bash
set -e

chmod +s /usr/bin/newuidmap
chmod +s /usr/bin/newgidmap

export VOLUME_DRIVER=overlay-xfs
export GROOTFS_TEST_UID=0
export GROOTFS_TEST_GID=0
export STORE=/var/vcap/data/grootfs/store/privileged

echo "I AM INTEGRATION: ${VOLUME_DRIVER} (${GROOTFS_TEST_UID}:${GROOTFS_TEST_GID})"

ginkgo -race -r integration
