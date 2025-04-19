from flask import Flask, request

from werkzeug.datastructures import ImmutableMultiDict

import json

import base64

import logging

from typing import List


"""
- Google stuff based upon [the service account key flow doco](https://developers.google.com/identity/protocols/oauth2/service-account#httprest_1).

Example invocation:

```bash
flask --app=test/python/stackql_test_tooling/flask/oauth/token_srv run --port=8070
```


Then, in some other shell:

```bash

curl -d 'grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Ajwt-bearer&assertion=eyJhIjogImIifQ==.eyJzdWIiOiAianMifQ==.ZXlKaElqb2dJbUlpZlE9PS5leUp6ZFdJaU9pQWlhbk1pZlE9PQ==' http://127.0.0.1:8070/google/simple/token
```
"""

app = Flask(__name__)

app.logger.setLevel(logging.INFO)

class GoogleServiceAccountJWT(object):

    def __init__(self, encoded_jwt: str) -> None:
        self._encoded_jwt = encoded_jwt
        split_jwt = encoded_jwt.split('.')
        if len(split_jwt) != 3:
            raise ValueError(f'invalid JWT; length {len(split_jwt)} != 3')
        self._jwt_header: dict = json.loads(base64.urlsafe_b64decode(self._pad(split_jwt[0])).decode('utf-8'))
        self._jwt_claims: dict = json.loads(base64.urlsafe_b64decode(self._pad(split_jwt[1])).decode('utf-8'))
        self._jwt_signature: bytes = base64.urlsafe_b64decode(self._pad(split_jwt[2]))
    
    def _pad(self, s: str) -> str:
        return s + '=' * (4 - len(s) % 4)

    def generate_google_response_dict(self, default_scopes: List[str]=["https://www.googleapis.com/auth/cloud-platform"]) -> dict:
        return {
            "access_token": "some-dummy-token",
            "scope": ' '.join(default_scopes),
            "token_type": "Bearer",
            "expires_in": 3600
        }

# conforming to [RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749#section-5.1)
_SIMPLE_RESPONSE = {
            "access_token": "some-dummy-token",
            "scope": 'my-scope',
            "token_type": "Bearer",
            "refresh_token": "some-dummy-refresh-token", # optional, per the RFC
            "expires_in": 3600
        }

def _form_to_str(form: ImmutableMultiDict) -> str:
    return json.dumps(form.to_dict(flat=False), sort_keys=True)

@app.route("/google/simple/token", methods=['POST'])
def google_simple_token():
    request_data = request.form
    proffrered_jwt = request_data['assertion']
    app.logger.info(f'proffered_jwt: {proffrered_jwt}')
    google_jwt: GoogleServiceAccountJWT = GoogleServiceAccountJWT(proffrered_jwt)
    return json.dumps(google_jwt.generate_google_response_dict(), sort_keys=True)

@app.route("/contrived/simple/error/token", methods=['POST'])
def google_simple_error_token():
    return json.dumps({"msg": "auth failed"}, sort_keys=True), 401

@app.route("/contrived/simple/token", methods=['POST'])
def contrived_simple_token():
    request_data = request.form
    app.logger.info(f'POST /contrived/simple/token request data: {_form_to_str(request_data)}')
    return json.dumps(_SIMPLE_RESPONSE, sort_keys=True)
