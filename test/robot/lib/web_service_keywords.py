from robot.api.deco import library, keyword

from robot.libraries.Process import Process

import json

from requests import get, post, Response

import os

from typing import Union, Tuple, List

@library
class web_service_keywords(Process):

    _DEFAULT_SQLITE_DB_PATH: str = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "tmp", "robot_cli_affirmation_store.db"))

    def _get_dsn(self) -> str:
        return self._DEFAULT_SQLITE_DB_PATH

    def __init__(self):
        self._affirmation_store_web_service = None
        self._web_server_app: str = 'test/python/flask/oauth2/token_srv'
        self._github_app: str = 'test/python/flask/github/app'
        self._tls_key_path: str = 'test/server/mtls/credentials/pg_server_key.pem'
        self._tls_cert_path: str = 'test/server/mtls/credentials/pg_server_cert.pem'
        super().__init__()

    @keyword
    def create_oauth2_client_credentials_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._web_server_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'token-client-credentials-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'token-client-credentials-{port}-stderr.txt'))
        )
    
    @keyword
    def create_github_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._github_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'github-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'github-server-{port}-stderr.txt'))
        )

    @keyword
    def send_get_request(
        self,
        address: str
    ) -> Response:
        """
        Send a simple get request.
        """
        return get(address)

    @keyword
    def send_json_post_request(
        self,
        address: str,
        input: dict
    ) -> Response:
        """
        Send a canonical json post request.
        """
        return post(address, json=input)
