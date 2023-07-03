

# This is simply a troubleshooting scratch pad

Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

choco install -y git

Import-Module $env:ChocolateyInstall\helpers\chocolateyProfile.psm1 

refreshenv

git config --global core.autocrlf false

choco install -y python3

refreshenv

choco install -y --force openjdk11

refreshenv

choco install -y maven

refreshenv

choco install -y --force mingw --version=8.1.0

refreshenv


pip install -r ./requirements.txt

$Version = convertfrom-stringdata (get-content ./version.txt -raw)
$BuildMajorVersion = $Version.'MajorVersion'
$BuildMinorVersion = $Version.'MinorVersion'

$env:BUILDMAJORVERSION = $BuildMajorVersion
$env:BUILDMINORVERSION = $BuildMinorVersion

echo "BUILDMAJORVERSION=$env:BUILDMAJORVERSION" >> $GITHUB_ENV
echo "BUILDMINORVERSION=$env:BUILDMINORVERSION" >> $GITHUB_ENV
echo "BUILDPATCHVERSION=$env:BUILDPATCHVERSION" >> $GITHUB_ENV

python cicd/python/build.py --verbose --build 


openssl req -x509 -keyout test/server/mtls/credentials/pg_server_key.pem -out test/server/mtls/credentials/pg_server_cert.pem -config test/server/mtls/openssl.cnf -days 365
openssl req -x509 -keyout test/server/mtls/credentials/pg_client_key.pem -out test/server/mtls/credentials/pg_client_cert.pem -config test/server/mtls/openssl.cnf -days 365
openssl req -x509 -keyout test/server/mtls/credentials/pg_rubbish_key.pem -out test/server/mtls/credentials/pg_rubbish_cert.pem -config test/server/mtls/openssl.cnf -days 365

$env:PSQL_EXE = C:\Program Files\PostgreSQL\13\bin\psql

robot -d test/robot/functional -t 'Left Outer Join Users' test/robot/functional


java  `
  '-Dfile.encoding=UTF-8' `
  "-Dmockserver.initializationJsonPath=${HOME}/stackql/test/mockserver/expectations/static-google-admin-expectations.json" `
   -jar `
   ${HOME}/stackql/test/downloads/mockserver-netty-5.12.0-shaded.jar  `
   -serverPort 1098 -logLevel INFO

java  `
   '-Dfile.encoding=UTF-8' `
   "-Dmockserver.initializationJsonPath=${HOME}/stackql/test/mockserver/expectations/static-aws-expectations.json" `
    -jar `
    ${HOME}/stackql/test/downloads/mockserver-netty-5.12.0-shaded.jar  `
    -serverPort 1091 -logLevel INFO

java  `
    '-Dfile.encoding=UTF-8' `
    "-Dmockserver.initializationJsonPath=${HOME}/stackql/test/mockserver/expectations/static-gcp-expectations.json" `
     -jar `
     ${HOME}/stackql/test/downloads/mockserver-netty-5.12.0-shaded.jar  `
     -serverPort 1080 -logLevel INFO

.\stackql.exe `
--registry='{"url": "file://C:/Users/krimmer/stackql/test/registry-mocked", "localDocRoot": "C:/Users/krimmer/stackql/test/registry-mocked", "verifyConfig": {"nopVerify": true}}' `
--tls.allowInsecure `
--auth='{"google": {"credentialsfilepath": "C:/Users/krimmer/stackql/test/assets/credentials/dummy/google/functional-test-dummy-sa-key.json", \"type\": "service_account"}, "googleadmin": {"credentialsfilepath": "C:/Users/krimmer/stackql/test/assets/credentials/dummy/google/functional-test-dummy-sa-key.json", \"type\": "service_account"}, "aws": {"type": "aws_signing_v4", "credentialsfilepath": "C:/Users/krimmer/stackql/test/assets/credentials/dummy/aws/functional-test-dummy-aws-key.txt", "keyID": "NON_SECRET"}}' `
exec "select aid.UserName as aws_user_name ,json_extract(gad.name, '$.fullName') as gcp_user_name ,lower(substr(aid.UserName, 1, 5)) as aws_fuzz_name ,lower(substr(json_extract(gad.name, '$.fullName'), 1, 5)) as gcp_fuzz_name from aws.iam.users aid LEFT OUTER JOIN googleadmin.directory.users gad ON lower(substr(aid.UserName, 1, 5)) = lower(substr(json_extract(gad.name, '$.fullName'), 1, 5)) WHERE aid.region = 'us-east-1' AND gad.domain = 'grubit.com' ORDER BY aws_user_name DESC ;"


