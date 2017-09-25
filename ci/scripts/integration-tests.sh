#!/bin/bash
set -e


args=$@
[ "$args" == "" ] && args="-r integration"
ginkgo -nodes 1 -p -race $args
