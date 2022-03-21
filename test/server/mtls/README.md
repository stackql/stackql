

## mTLS setup for stackql server

### Prepare tls collateral

```bash

openssl req -x509 -keyout ./credentials/pg_server_key.pem -out ./credentials/pg_server_cert.pem -config ./openssl.cnf -days 365

openssl req -x509 -keyout ./credentials/pg_client_key.pem -out ./credentials/pg_client_cert.pem -config ./openssl.cnf -days 365


export CLIENT_CERT=$(base64 ./credentials/pg_client_cert.pem)

```

### Run

```
./build/pg-srv-lite srv --address=0.0.0.0 --port=5444 --tlsconfig='{ "keyFilePath": "tls/secrets/pg_server_key.pem", "certFilePath": "tls/secrets/pg_server_cert.pem", "clientCAs": [ "'${CLIENT_CERT}'" ] }'
```

### Test with psql client

`psql "host=127.0.0.1 port=5444 user=myuser dbname=mydatabase sslmode=verify-full sslcert=tls/secrets/pg_client_cert.pem sslkey=tls/secrets/pg_client_key.pem sslrootcert=tls/secrets/pg_server_cert.pem"`