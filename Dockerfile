FROM golang:1.21.5-bullseye AS sourceprep

ENV SRC_DIR=/work/stackql/src

ENV BUILD_DIR=/work/stackql/build

RUN mkdir -p ${SRC_DIR} ${BUILD_DIR}

ADD internal  ${SRC_DIR}/internal

ADD pkg ${SRC_DIR}/pkg

ADD stackql ${SRC_DIR}/stackql

ADD test ${SRC_DIR}/test

COPY go.mod go.sum ${SRC_DIR}/

RUN  cd ${SRC_DIR} && ls && go get -v -t -d ./...

FROM sourceprep AS builder 

ARG BUILDMAJORVERSION="1"
ARG BUILDMINORVERSION="1"
ARG BUILDPATCHVERSION="1"
ARG BUILDCOMMITSHA="1"
ARG BUILDSHORTCOMMITSHA="1"
ARG BUILDDATE="1"
ARG PLANCACHEENABLED="1"
ARG BUILDPLATFORM="1"
ARG RUN_INTEGRATION_TESTS="1"

ENV BUILDMAJORVERSION=${BUILDMAJORVERSION}
ENV BUILDMINORVERSION=${BUILDMINORVERSION}
ENV BUILDPATCHVERSION=${BUILDPATCHVERSION}
ENV BUILDCOMMITSHA=${BUILDCOMMITSHA}
ENV BUILDSHORTCOMMITSHA=${BUILDSHORTCOMMITSHA}
ENV BUILDDATE=${BUILDDATE}
ENV PLANCACHEENABLED=${PLANCACHEENABLED}
ENV BUILDPLATFORM=${BUILDPLATFORM}

ENV SRC_DIR=/work/stackql/src

ENV BUILD_DIR=/work/stackql/build

RUN   cd ${SRC_DIR} \
      && go test --tags "sqlite_stackql" ./... \
      && go build -ldflags "-X github.com/stackql/stackql/internal/stackql/cmd.BuildMajorVersion=$BUILDMAJORVERSION \
          -X github.com/stackql/stackql/internal/stackql/cmd.BuildMinorVersion=$BUILDMINORVERSION \
          -X github.com/stackql/stackql/internal/stackql/cmd.BuildPatchVersion=$BUILDPATCHVERSION \
          -X github.com/stackql/stackql/internal/stackql/cmd.BuildCommitSHA=$BUILDCOMMITSHA \
          -X github.com/stackql/stackql/internal/stackql/cmd.BuildShortCommitSHA=$BUILDSHORTCOMMITSHA \
          -X \"github.com/stackql/stackql/internal/stackql/cmd.BuildDate=$BUILDDATE\" \
          -X \"stackql/internal/stackql/planbuilder.PlanCacheEnabled=$PLANCACHEENABLED\" \
          -X github.com/stackql/stackql/internal/stackql/cmd.BuildPlatform=$BUILDPLATFORM" \
        --tags "sqlite_stackql" \
        -o ${BUILD_DIR}/stackql ./stackql

FROM python:3.11-bullseye AS utility

ARG TEST_ROOT_DIR=/opt/test/stackql

ARG RUN_INTEGRATION_TESTS

ENV TEST_ROOT_DIR=${TEST_ROOT_DIR}

RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
      openssl \
      postgresql-client \
      sqlite3

FROM utility AS certificates

ARG TEST_ROOT_DIR=/opt/test/stackql

ARG RUN_INTEGRATION_TESTS

ENV TEST_ROOT_DIR=${TEST_ROOT_DIR}

RUN mkdir -p ${TEST_ROOT_DIR}

ADD test ${TEST_ROOT_DIR}/test

RUN openssl req -x509 -keyout ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_server_key.pem -out  ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_server_cert.pem  -config ${TEST_ROOT_DIR}/test/server/mtls/openssl.cnf -days 365 \
    && openssl req -x509 -keyout ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_client_key.pem -out  ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_client_cert.pem  -config ${TEST_ROOT_DIR}/test/server/mtls/openssl.cnf -days 365 \
    && openssl req -x509 -keyout ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_rubbish_key.pem -out ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_rubbish_cert.pem -config ${TEST_ROOT_DIR}/test/server/mtls/openssl.cnf -days 365

FROM python:3.11-bullseye AS registrymock

ARG TEST_ROOT_DIR=/opt/test/stackql

ARG RUN_INTEGRATION_TESTS

ENV TEST_ROOT_DIR=${TEST_ROOT_DIR}

RUN mkdir -p ${TEST_ROOT_DIR}

ADD cicd ${TEST_ROOT_DIR}/cicd

COPY --from=certificates /opt/test/stackql ${TEST_ROOT_DIR}/

RUN pip3 install -r ${TEST_ROOT_DIR}/cicd/requirements.txt \
    && python3 ${TEST_ROOT_DIR}/test/python/registry-rewrite.py

FROM utility AS integration

ARG TEST_ROOT_DIR=/opt/test/stackql

ENV TEST_ROOT_DIR=${TEST_ROOT_DIR}

ARG BUILDMAJORVERSION="1"
ARG BUILDMINORVERSION="1"
ARG BUILDPATCHVERSION="1"
ARG BUILDCOMMITSHA="1"
ARG BUILDSHORTCOMMITSHA="1"
ARG BUILDDATE="1"
ARG PLANCACHEENABLED="1"
ARG BUILDPLATFORM="1"
ARG RUN_INTEGRATION_TESTS

ENV BUILDMAJORVERSION=${BUILDMAJORVERSION}
ENV BUILDMINORVERSION=${BUILDMINORVERSION}
ENV BUILDPATCHVERSION=${BUILDPATCHVERSION}
ENV BUILDCOMMITSHA=${BUILDCOMMITSHA}
ENV BUILDSHORTCOMMITSHA=${BUILDSHORTCOMMITSHA}
ENV BUILDDATE=${BUILDDATE}
ENV PLANCACHEENABLED=${PLANCACHEENABLED}
ENV BUILDPLATFORM=${BUILDPLATFORM}

RUN mkdir -p ${TEST_ROOT_DIR}/build

COPY --from=registrymock /opt/test/stackql ${TEST_ROOT_DIR}/

COPY --from=builder /work/stackql/build/stackql ${TEST_ROOT_DIR}/build/

RUN  if [ "${RUN_INTEGRATION_TESTS}" = "1" ]; then robot ${TEST_ROOT_DIR}/test/robot/functional; fi

FROM ubuntu:22.04 AS app

ARG TEST_ROOT_DIR=/opt/test/stackql

ENV TEST_ROOT_DIR=${TEST_ROOT_DIR}

ARG APP_DIR=/srv/stackql

ARG STACKQL_CFG_ROOT=/opt/stackql

ARG STACKQL_PG_PORT=5477

ENV APP_DIR="${APP_DIR}"

ENV STACKQL_CFG_ROOT="${STACKQL_CFG_ROOT}"

ENV STACKQL_PG_PORT="${STACKQL_PG_PORT}"

RUN mkdir -p ${APP_DIR} ${STACKQL_CFG_ROOT}/keys ${STACKQL_CFG_ROOT}/srv/credentials ${STACKQL_CFG_ROOT}/credentials/dummy ${STACKQL_CFG_ROOT}/registry ${STACKQL_CFG_ROOT}/logs ${STACKQL_CFG_ROOT}/db

ENV PATH="${APP_DIR}:${PATH}"

COPY --from=integration ${TEST_ROOT_DIR}/build/stackql ${APP_DIR}/

RUN apt-get update \
    && apt-get install -y ca-certificates \
    && update-ca-certificates

EXPOSE ${STACKQL_PG_PORT}/tcp

WORKDIR ${STACKQL_CFG_ROOT}

CMD ["/bin/bash", "-c", "stackql --pgsrv.port=${STACKQL_PG_PORT} srv"]
