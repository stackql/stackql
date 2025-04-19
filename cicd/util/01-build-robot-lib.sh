#! /usr/bin/env bash

poetryExe="$(which poetry)"
rv="$?"
if [ $rv -ne 0 ]; then
    >&2 echo "Poetry is not installed. Please install it first." 
    exit 1
fi
if [ "$poetryExe" = "" ]; then
    >&2 echo "No poetry executable found in PATH. Please install it first."
    exit 1
fi

CURDIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

REPOSITORY_ROOT="$(realpath ${CURDIR}/../..)"

PACKAGE_ROOT="${REPOSITORY_ROOT}/test"

venv_path="${REPOSITORY_ROOT}/.venv"


rm -f ${PACKAGE_ROOT}/dist/*.whl || true

cd "${PACKAGE_ROOT}"

poetry install

poetry build

filez="$(ls ${PACKAGE_ROOT}/dist/*.whl)" || true

if [ "${filez}" = "" ]; then
    >&2 echo "No wheel files found in ${PACKAGE_ROOT}/dist. Please check the build process."
    exit 1
else
    echo "Wheel files found in ${PACKAGE_ROOT}/dist: ${filez}"
fi


# >&2 echo "Artifact built successfully: ${expectedRobotLibArtifact}"






