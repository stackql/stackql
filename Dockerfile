FROM golang:1.18.4-bullseye AS builder

ENV SRC_DIR=/work/stackql/src

ENV BUILD_DIR=/work/stackql/build

RUN mkdir -p ${SRC_DIR} ${BUILD_DIR}

ADD internal  ${SRC_DIR}/internal

ADD pkg ${SRC_DIR}/pkg

ADD stackql ${SRC_DIR}/stackql

ADD test ${SRC_DIR}/test

COPY go.mod go.sum ${SRC_DIR}/

RUN  cd ${SRC_DIR} && ls && go get -v -t -d ./... && go test --tags "json1" ./... \
     && go build --tags "json1" -o ${BUILD_DIR}/stackql ./stackql

FROM ubuntu:22.04 AS certificates

ARG TEST_ROOT_DIR=/opt/test/stackql

ENV TEST_ROOT_DIR=${TEST_ROOT_DIR}

RUN mkdir -p ${TEST_ROOT_DIR}

COPY --from=builder /work/stackql/src ${TEST_ROOT_DIR}/

RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
      openssl \
    && openssl req -x509 -keyout ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_server_key.pem -out  ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_server_cert.pem  -config ${TEST_ROOT_DIR}/test/server/mtls/openssl.cnf -days 365 \
    && openssl req -x509 -keyout ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_client_key.pem -out  ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_client_cert.pem  -config ${TEST_ROOT_DIR}/test/server/mtls/openssl.cnf -days 365 \
    && openssl req -x509 -keyout ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_rubbish_key.pem -out ${TEST_ROOT_DIR}/test/server/mtls/credentials/pg_rubbish_cert.pem -config ${TEST_ROOT_DIR}/test/server/mtls/openssl.cnf -days 365

FROM ubuntu:22.04 AS registrymock

ARG TEST_ROOT_DIR=/opt/test/stackql

ENV TEST_ROOT_DIR=${TEST_ROOT_DIR}

RUN mkdir -p ${TEST_ROOT_DIR}

COPY --from=certificates /opt/test/stackql ${TEST_ROOT_DIR}/

RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
      python3 \
      python3-pip \
    && pip3 install PyYaml \
    && python3 ${TEST_ROOT_DIR}/test/python/registry-rewrite.py

FROM ubuntu:22.04 AS integration

ARG TEST_ROOT_DIR=/opt/test/stackql

ENV TEST_ROOT_DIR=${TEST_ROOT_DIR}

RUN mkdir -p ${TEST_ROOT_DIR}/build

COPY --from=registrymock /opt/test/stackql ${TEST_ROOT_DIR}/

COPY --from=builder /work/stackql/build/stackql ${TEST_ROOT_DIR}/build/

RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
      default-jdk \
      default-jre \
      maven \
      openssl \
      postgresql-client \
      python3 \
      python3-pip \
    && pip3 install PyYaml robotframework \
    && mvn \
        org.apache.maven.plugins:maven-dependency-plugin:3.0.2:copy \
        -Dartifact=org.mock-server:mockserver-netty:5.12.0:jar:shaded \
        -DoutputDirectory=${TEST_ROOT_DIR}/test/downloads \
    && robot ${TEST_ROOT_DIR}/test/robot/functional

FROM ubuntu:22.04 AS app

ARG TEST_ROOT_DIR=/opt/test/stackql

ENV TEST_ROOT_DIR=${TEST_ROOT_DIR}

ARG APP_DIR=/srv/stackql

ARG STACKQL_CFG_ROOT=/opt/stackql

ARG STACKQL_PG_PORT=5477

ENV APP_DIR="${APP_DIR}"

ENV STACKQL_CFG_ROOT="${STACKQL_CFG_ROOT}"

ENV STACKQL_PG_PORT="${STACKQL_PG_PORT}"

RUN mkdir -p ${APP_DIR} ${STACKQL_CFG_ROOT}/keys ${STACKQL_CFG_ROOT}/srv/credentials ${STACKQL_CFG_ROOT}/credentials/dummy ${STACKQL_CFG_ROOT}/registry

ENV PATH="${APP_DIR}:${PATH}"

COPY --from=integration ${TEST_ROOT_DIR}/build/stackql ${APP_DIR}/

RUN apt-get update \
    && apt-get install -y ca-certificates \
    && update-ca-certificates

EXPOSE ${STACKQL_PG_PORT}/tcp

WORKDIR ${STACKQL_CFG_ROOT}

CMD ["/bin/bash", "-c", "stackql --pgsrv.port=${STACKQL_PG_PORT} srv"]


