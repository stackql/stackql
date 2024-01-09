

## mTLS setup for stackql server

We have implemented strict TLS where the server is started in `RequireAndVerifyClientCert` mode, which is the default and (at present) only option where server TLS config is supplied.  Clients attempting to access with `sslmode=disable` will be rejected.  For more detail on client `sslmode` options, please see [the __`libpq`__ doco](https://www.postgresql.org/docs/8.4/libpq-connect.html#LIBPQ-CONNECT-SSLMODE).

### Prepare tls collateral

```bash

openssl req -x509 -keyout ./cicd/vol/srv/credentials/pg_server_key.pem -out ./cicd/vol/srv/credentials/pg_server_cert.pem -config ./openssl.cnf -days 365

openssl req -x509 -keyout ./cicd/vol/srv/credentials/pg_client_key.pem -out ./cicd/vol/srv/credentials/pg_client_cert.pem -config ./openssl.cnf -days 365


export CLIENT_CERT=$(base64 ./cicd/vol/srv/credentials/pg_client_cert.pem)

echo "$CLIENT_CERT" > ./cicd/vol/srv/credentials/pg_client_cert.pem.base64

```

### Run

```
./build/stackql srv --pgsrv.address=0.0.0.0 --pgsrv.port=5444 --pgsrv.tls='{ "keyFilePath": "./cicd/vol/srv/credentials/pg_server_key.pem", "certFilePath": "./cicd/vol/srv/credentials/pg_server_cert.pem", "clientCAs": [ "'${CLIENT_CERT}'" ] }'
```

### Test with psql client

`psql "host=127.0.0.1 port=5444 user=myuser dbname=mydatabase sslmode=verify-full sslcert=./cicd/vol/srv/credentials/pg_client_cert.pem sslkey=./cicd/vol/srv/credentials/pg_client_key.pem sslrootcert=./cicd/vol/srv/credentials/pg_server_cert.pem"`
