#!/usr/bin/env bash

CUR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-${(%):-%x}}")" && pwd)"

export REPOSITORY_ROOT="$(realpath ${CUR_DIR}/../..)"



