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
        self._gcp_app: str = 'test/python/flask/gcp/app'
        self._okta_app: str = 'test/python/flask/okta/app'
        self._static_auth_testing_app: str = 'test/python/flask/static_auth/app'
        self._aws_app: str = 'test/python/flask/aws/app'
        self._azure_app: str = 'test/python/flask/azure/app'
        self._digitalocean_app: str = 'test/python/flask/digitalocean/app'
        self._googleadmin_app: str = 'test/python/flask/googleadmin/app'
        self._k8s_app: str = 'test/python/flask/k8s/app'
        self._registry_app: str = 'test/python/flask/registry/app'
        self._sumologic_app: str = 'test/python/flask/sumologic/app'
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
    def create_gcp_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._gcp_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'gcp-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'gcp-server-{port}-stderr.txt'))
        )
    
    @keyword
    def create_okta_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._okta_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'okta-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'okta-server-{port}-stderr.txt'))
        )
    
    @keyword
    def create_aws_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._aws_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'aws-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'aws-server-{port}-stderr.txt'))
        )
    
    @keyword
    def create_static_auth_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._static_auth_testing_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'static-auth-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'static-auth-server-{port}-stderr.txt'))
        )
    
    @keyword
    def create_google_admin_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._googleadmin_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'google-admin-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'google-admin-server-{port}-stderr.txt'))
        )
    
    @keyword
    def create_k8s_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._k8s_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'k8s-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'k8s-server-{port}-stderr.txt'))
        )
    
    @keyword
    def create_registry_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._registry_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            # f'--cert={self._tls_cert_path}',
            # f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'registry-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'registry-server-{port}-stderr.txt'))
        )
    
    @keyword
    def create_azure_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._azure_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'azure-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'azure-server-{port}-stderr.txt'))
        )
    
    @keyword
    def create_sumologic_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._sumologic_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'sumologic-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'sumologic-server-{port}-stderr.txt'))
        )
    
    @keyword
    def create_digitalocean_web_service(
        self,
        port: int,
        host: str = '0.0.0.0'
    ) -> None:
        """
        Sign the input.
        """
        return self.start_process(
            'flask',
            f'--app={self._digitalocean_app}',
            'run',
            f'--host={host}', # generally, `0.0.0.0`; otherwise, invisible on `docker.host.internal` etc
            f'--port={port}',
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'digitalocean-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(os.path.dirname(__file__), '..', 'log', f'digitalocean-server-{port}-stderr.txt'))
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
