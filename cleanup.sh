#!/bin/bash

for m in $(cat /proc/mounts | grep overlay | cut -d ' ' -f 2); do umount $m; done
rm -rf /var/vcap/data/grootfs/store/privileged/store/
