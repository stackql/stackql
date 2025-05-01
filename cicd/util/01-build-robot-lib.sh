#! /usr/bin/env bash

checkPoetry () {
    if ! command -v poetry &> /dev/null
    then
        >&2 echo "Poetry is not installed. Please install it first." 
        exit 1
    fi
}

CURDIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

REPOSITORY_ROOT="$(realpath ${CURDIR}/../..)"

PACKAGE_ROOT="${REPOSITORY_ROOT}/test"

rm -f ${PACKAGE_ROOT}/dist/*.whl || true

if [ ! -d "${PACKAGE_ROOT}/.venv" ]; then
  >&2 echo "No existing virtual environment, creating one..."
  >&2 echo "Creating virtual environment in ${PACKAGE_ROOT}/.venv"
  python -m venv "${PACKAGE_ROOT}/.venv"
  >&2 echo "Virtual environment created."
  >&2 echo "Installing poetry into virtual environment."
  ${PACKAGE_ROOT}/.venv/bin/pip install -U pip setuptools
  ${PACKAGE_ROOT}/.venv/bin/pip install poetry
  >&2 echo "Poetry installed into virtual environment."
else
  >&2 echo "Using existing virtual environment in ${PACKAGE_ROOT}/.venv"
fi

cd "${PACKAGE_ROOT}"

source ${PACKAGE_ROOT}/.venv/bin/activate

checkPoetry

poetry install

poetry build

filez="$(ls ${PACKAGE_ROOT}/dist/*.whl)" || true

if [ "${filez}" = "" ]; then
    >&2 echo "No wheel files found in ${PACKAGE_ROOT}/dist. Please check the build process."
    exit 1
else
    echo "Wheel files found in ${PACKAGE_ROOT}/dist: ${filez}"
fi








