

## mTLS setup for stackql server

### Prepare tls collateral

```bash

openssl req -x509 -keyout ./vol/srv/credentials/pg_server_key.pem -out ./vol/srv/credentials/pg_server_cert.pem -config ./openssl.cnf -days 365

openssl req -x509 -keyout ./vol/srv/credentials/pg_client_key.pem -out ./vol/srv/credentials/pg_client_cert.pem -config ./openssl.cnf -days 365


export CLIENT_CERT=$(base64 ./vol/srv/credentials/pg_client_cert.pem)

```

### Run

```
./build/stackql srv --pgsrv.address=0.0.0.0 --pgsrv.port=5444 --pgsrv.tls='{ "keyFilePath": "./vol/srv/credentials/pg_server_key.pem", "certFilePath": "./vol/srv/credentials/pg_server_cert.pem", "clientCAs": [ "'${CLIENT_CERT}'" ] }'
```

### Test with psql client

`psql "host=127.0.0.1 port=5444 user=myuser dbname=mydatabase sslmode=verify-full sslcert=./vol/srv/credentials/pg_client_cert.pem sslkey=./vol/srv/credentials/pg_client_key.pem sslrootcert=./vol/srv/credentials/pg_server_cert.pem"`