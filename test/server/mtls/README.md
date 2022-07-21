

## mTLS setup for stackql server

### Prepare tls collateral

```bash

openssl req -x509 -keyout ./credentials/pg_server_key.pem -out ./credentials/pg_server_cert.pem -config ./openssl.cnf -days 365

openssl req -x509 -keyout ./credentials/pg_client_key.pem -out ./credentials/pg_client_cert.pem -config ./openssl.cnf -days 365


export CLIENT_CERT=$(base64 ./credentials/pg_client_cert.pem)

```

### Run

```
stackql srv --pgsrv.address=0.0.0.0 --pgsrv.port=5444 --pgsrv.tls='{ "keyFilePath": "credentials/pg_server_key.pem", "certFilePath": "credentials/pg_server_cert.pem", "clientCAs": [ "'${CLIENT_CERT}'" ] }'
```

### Test with psql client

`psql "host=127.0.0.1 port=5444 user=myuser dbname=mydatabase sslmode=verify-full sslcert=credentials/pg_client_cert.pem sslkey=credentials/pg_client_key.pem sslrootcert=credentials/pg_server_cert.pem"`