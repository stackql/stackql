#! /usr/bin/env bash

CURDIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

REPOSITORY_ROOT="$(realpath ${CURDIR}/../..)"

PACKAGE_ROOT="${REPOSITORY_ROOT}/test"

venv_path="${REPOSITORY_ROOT}/.venv"

CURDIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

REPOSITORY_ROOT="$(realpath ${CURDIR}/../..)"

venv_path="${REPOSITORY_ROOT}/.venv"

# expectedRobotLibArtifact="$(realpath ${PACKAGE_ROOT}/dist/stackql_test_tooling-0.1.0-py3-none-any.whl)"

if [ ! -d "${venv_path}" ]; then
    echo "Creating virtual environment at ${venv_path}"
    python3 -m venv ${venv_path}
else
    echo "Virtual environment already exists at ${venv_path}"
fi

filez="$(ls ${PACKAGE_ROOT}/dist/*.whl)" || true

if [ "${filez}" = "" ]; then
    >&2 echo "No wheel files found in ${PACKAGE_ROOT}/dist. Please check the build process."
    exit 1
else
    echo "Wheel files found in ${PACKAGE_ROOT}/dist: ${filez}"
fi

source ${REPOSITORY_ROOT}/.venv/bin/activate

pip install -r ${REPOSITORY_ROOT}/cicd/requirements.txt

for file in ${PACKAGE_ROOT}/dist/*.whl; do
    pip3 install "$file" --force-reinstall
done








