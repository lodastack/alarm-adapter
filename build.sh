#!/bin/bash

set -e

# build bin file
make build

# make my dir
mv cmd/alarm-adapter/alarm-adapter $BUILD_ROOT