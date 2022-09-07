
cmake_minimum_required(VERSION 3.0)
project(stackql)

set(TARGET stackql_build)
set(EXECUTABLE stackql)

set(GOLANG_SRC_PATH_STR stackql)
set(GOLANG_BUILD_PATH_STR build)

set(OS_STR_MAC darwin)
set(OS_STR_LINUX linux)
set(OS_STR_WINDOWS windows)

set(EXE_SUFFIX_MAC _${OS_STR_MAC})
set(EXE_SUFFIX_LINUX _${OS_STR_LINUX})
set(EXE_SUFFIX_WINDOWS .exe)

set(BUILDMAJORVERSION 1)
set(BUILDMINORVERSION 1)
set(BUILDPATCHVERSION 1)
set(BUILDCOMMITSHA local-dev)
set(BUILDSHORTCOMMITSHA local-dev)
set(BUILDDATE 1900-01-01)

set(PLANCACHEENABLED true)

set(WINCOMPILER "CC=x86_64-w64-mingw32-gcc")
set(LINUXCOMPILER "CC=x86_64-linux-musl-gcc")

set(INT_TEST_DB_SRC_FILE "test/db/db.sqlite")
set(INT_TEST_DB_DEST_FILE "test/db/tmp/python-tests-tmp-db.sqlite")

enable_testing()

find_program(Python_EXECUTABLE python3)
find_program(Java_EXECUTABLE java)
find_program(Robot_EXECUTABLE robot)
find_program(Maven_EXECUTABLE mvn)

set(TEST_SCRIPT ${CMAKE_CURRENT_SOURCE_DIR}/test/python/main.py)
set(REG_RW_SCRIPT ${CMAKE_CURRENT_SOURCE_DIR}/test/python/registry-rewrite.py)
set(ROBOT_TEST_DIR_PATH ${CMAKE_CURRENT_SOURCE_DIR}/test/robot/functional)

add_custom_target(${TARGET} ALL)


if(NOT Maven_EXECUTABLE)
    message(FATAL_ERROR "Cannot find the mvn executable!")
endif()

if(NOT Python_EXECUTABLE)
    message(FATAL_ERROR "Cannot find the python3 executable!")
endif()

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND mvn org.apache.maven.plugins:maven-dependency-plugin:3.2.0:copy -Dartifact=org.mock-server:mockserver-netty:5.12.0:jar:shaded -DoutputDirectory=${CMAKE_CURRENT_SOURCE_DIR}/test/downloads -DdestFileName=mockserver-netty.jar -DoverWrite=true
  COMMENT "maven copy mock server artifact"
)

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND ${Python_EXECUTABLE} ${REG_RW_SCRIPT}
  COMMENT "set up dummy regsitry for server simulations"
)

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND
    openssl req -x509 -keyout test/server/mtls/credentials/pg_server_key.pem -out test/server/mtls/credentials/pg_server_cert.pem -config test/server/mtls/openssl.cnf -days 365
  COMMENT "set up server TLS credentials for robot tests"
)

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND
    openssl req -x509 -keyout test/server/mtls/credentials/pg_client_key.pem -out test/server/mtls/credentials/pg_client_cert.pem -config test/server/mtls/openssl.cnf -days 365
  COMMENT "set up client TLS credentials for robot tests"
)

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND
    openssl req -x509 -keyout test/server/mtls/credentials/pg_rubbish_key.pem -out test/server/mtls/credentials/pg_rubbish_cert.pem -config test/server/mtls/openssl.cnf -days 365 
  COMMENT "set up rubbish TLS credentials for robot tests"
)

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND go test -timeout 2400s -p 2 --tags "json1" ./...
  COMMENT "Running Go Unit Tests"
)

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND ${CMAKE_COMMAND} -E copy ${INT_TEST_DB_SRC_FILE} ${INT_TEST_DB_DEST_FILE}
  COMMENT "Preparing for integration tests; copy db starting point file"
)

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND go build  --tags "json1" -ldflags
  "-X stackql/internal/stackql/cmd.BuildMajorVersion=${BUILDMAJORVERSION} \
  -X stackql/internal/stackql/cmd.BuildMinorVersion=${BUILDMINORVERSION} \
  -X stackql/internal/stackql/cmd.BuildPatchVersion=${BUILDPATCHVERSION} \
  -X stackql/internal/stackql/cmd.BuildCommitSHA=${BUILDCOMMITSHA} \
  -X stackql/internal/stackql/cmd.BuildShortCommitSHA=${BUILDSHORTCOMMITSHA} \
  -X \"stackql/internal/stackql/cmd.BuildDate=${BUILDDATE}\" \
  -X \"stackql/internal/stackql/planbuilder.PlanCacheEnabled=${PLANCACHEENABLED}\" \
  -X stackql/internal/stackql/cmd.BuildPlatform=$ENV{GOOS}"
  -o "${CMAKE_CURRENT_SOURCE_DIR}/${GOLANG_BUILD_PATH_STR}/${EXECUTABLE}"
  "${CMAKE_CURRENT_SOURCE_DIR}/${GOLANG_SRC_PATH_STR}"
  COMMENT "Building Go executable for local platform: $ENV{GOOS}"
)

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND ${CMAKE_COMMAND} -E env GOOS=${OS_STR_MAC} go build --tags "json1" -ldflags
  "-X stackql/internal/stackql/cmd.BuildMajorVersion=${BUILDMAJORVERSION} \
  -X stackql/internal/stackql/cmd.BuildMinorVersion=${BUILDMINORVERSION} \
  -X stackql/internal/stackql/cmd.BuildPatchVersion=${BUILDPATCHVERSION} \
  -X stackql/internal/stackql/cmd.BuildCommitSHA=${BUILDCOMMITSHA} \
  -X stackql/internal/stackql/cmd.BuildShortCommitSHA=${BUILDSHORTCOMMITSHA} \
  -X \"stackql/internal/stackql/cmd.BuildDate=${BUILDDATE}\" \
  -X \"stackql/internal/stackql/planbuilder.PlanCacheEnabled=${PLANCACHEENABLED}\" \
  -X stackql/internal/stackql/cmd.BuildPlatform=${OS_STR_MAC}"
  -o "${CMAKE_CURRENT_SOURCE_DIR}/${GOLANG_BUILD_PATH_STR}/${EXECUTABLE}${EXE_SUFFIX_MAC}"
  "${CMAKE_CURRENT_SOURCE_DIR}/${GOLANG_SRC_PATH_STR}"
  COMMENT "Building Go executable for ${OS_STR_MAC}"
)

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND ${CMAKE_COMMAND} -E env GOOS=${OS_STR_WINDOWS} ${WINCOMPILER} go build --tags "json1" -ldflags
  "-X stackql/internal/stackql/cmd.BuildMajorVersion=${BUILDMAJORVERSION} \
  -X stackql/internal/stackql/cmd.BuildMinorVersion=${BUILDMINORVERSION} \
  -X stackql/internal/stackql/cmd.BuildPatchVersion=${BUILDPATCHVERSION} \
  -X stackql/internal/stackql/cmd.BuildCommitSHA=${BUILDCOMMITSHA} \
  -X stackql/internal/stackql/cmd.BuildShortCommitSHA=${BUILDSHORTCOMMITSHA} \
  -X \"stackql/internal/stackql/cmd.BuildDate=${BUILDDATE}\" \
  -X \"stackql/internal/stackql/planbuilder.PlanCacheEnabled=${PLANCACHEENABLED}\" \
  -X stackql/internal/stackql/cmd.BuildPlatform=${OS_STR_WINDOWS}"
  -o "${CMAKE_CURRENT_SOURCE_DIR}/${GOLANG_BUILD_PATH_STR}/${EXECUTABLE}${EXE_SUFFIX_WINDOWS}"
  "${CMAKE_CURRENT_SOURCE_DIR}/${GOLANG_SRC_PATH_STR}"
  COMMENT "Building Go executable for ${OS_STR_WINDOWS}"
)

add_custom_command(TARGET ${TARGET}
  PRE_BUILD
  DEPENDS ${SRCS}
  WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
  COMMAND ${CMAKE_COMMAND} -E env GOOS=${OS_STR_LINUX} ${LINUXCOMPILER} go build -ldflags
  "-X stackql/internal/stackql/cmd.BuildMajorVersion=${BUILDMAJORVERSION} \
  -X stackql/internal/stackql/cmd.BuildMinorVersion=${BUILDMINORVERSION} \
  -X stackql/internal/stackql/cmd.BuildPatchVersion=${BUILDPATCHVERSION} \
  -X stackql/internal/stackql/cmd.BuildCommitSHA=${BUILDCOMMITSHA} \
  -X stackql/internal/stackql/cmd.BuildShortCommitSHA=${BUILDSHORTCOMMITSHA} \
  -X \"stackql/internal/stackql/cmd.BuildDate=${BUILDDATE}\" \
  -X \"stackql/internal/stackql/planbuilder.PlanCacheEnabled=${PLANCACHEENABLED}\" \
  -X stackql/internal/stackql/cmd.BuildPlatform=${OS_STR_LINUX}"
  -o "${CMAKE_CURRENT_SOURCE_DIR}/${GOLANG_BUILD_PATH_STR}/${EXECUTABLE}${EXE_SUFFIX_LINUX}"
  "${CMAKE_CURRENT_SOURCE_DIR}/${GOLANG_SRC_PATH_STR}"
  COMMENT "Building Go executable for ${OS_STR_LINUX}"
)


if(DEFINED ENV{STACKQL_TEST_LOG_LEVEL})
  set(STACKQL_TEST_LOG_LEVEL $ENV{STACKQL_TEST_LOG_LEVEL})
else()
  set(STACKQL_TEST_LOG_LEVEL warn)
endif()

if(DEFINED LIVE_INTEGRATION_TESTS)
  message(STATUS "LIVE_INTEGRATION_TESTS cmake variable = '${LIVE_INTEGRATION_TESTS}'")
else()
  set(LIVE_INTEGRATION_TESTS "")
  message(STATUS "LIVE_INTEGRATION_TESTS cmake variable is empty, only 'always' tests will be run")
endif()

add_test(NAME int-tests
    COMMAND ${Python_EXECUTABLE} ${TEST_SCRIPT} --additionalintegrationtestdir=${LIVE_INTEGRATION_TESTS} --loglevel=${STACKQL_TEST_LOG_LEVEL}
)

if(NOT Java_EXECUTABLE)
    message(FATAL_ERROR "Cannot find the java executable!")
endif()
if(NOT Robot_EXECUTABLE)
    message(FATAL_ERROR "Cannot find the robot executable!")
endif()
add_test(NAME robot-tests
    COMMAND ${Robot_EXECUTABLE} ${ROBOT_TEST_DIR_PATH}
)

add_custom_command(TARGET ${TARGET}
  POST_BUILD
  COMMAND ctest --verbose)


