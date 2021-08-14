#!/bin/bash

set -ex

pushd agent
go build
popd

pushd cli
go build
popd

pushd server
go build
popd
