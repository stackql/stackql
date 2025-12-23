from robot.api.deco import library, keyword

from robot.libraries.Process import Process

from requests import get, post, Response

import os

from pathlib import Path, PurePosixPath

from typing import Optional

@library
class web_service_keywords(Process):

    _THIS_DIR: str = PurePosixPath(Path(os.path.dirname(os.path.abspath(__file__)))).as_posix()
    
    _DEFAULT_APP_ROOT: str = os.path.join(_THIS_DIR, 'flask')

    _DEFAULT_TLS_KEY_PATH: str = 'test/server/mtls/credentials/pg_server_key.pem'

    _DEFAULT_TLS_CERT_PATH: str = 'test/server/mtls/credentials/pg_server_cert.pem'

    _DEFAULT_MOCKSERVER_PORT_GOOGLE                         = 1080
    _DEFAULT_MOCKSERVER_PORT_GOOGLEADMIN                    = 1098
    _DEFAULT_MOCKSERVER_PORT_STACKQL_AUTH_TESTING           = 1170
    _DEFAULT_MOCKSERVER_PORT_OKTA                           = 1090
    _DEFAULT_MOCKSERVER_PORT_AWS                            = 1091
    _DEFAULT_MOCKSERVER_PORT_K8S                            = 1092
    _DEFAULT_MOCKSERVER_PORT_GITHUB                         = 1093
    _DEFAULT_MOCKSERVER_PORT_AZURE                          = 1095
    _DEFAULT_MOCKSERVER_PORT_SUMOLOGIC                      = 1096
    _DEFAULT_MOCKSERVER_PORT_DIGITALOCEAN                   = 1097
    _DEFAULT_MOCKSERVER_PORT_OAUTH_CLIENT_CREDENTIALS_TOKEN = 2091
    _DEFAULT_MOCKSERVER_PORT_REGISTRY                       = 1094

    def _get_dsn(self) -> str:
        return self._sqlite_db_path

    def __init__(
        self,
        cwd: str,
        log_root: Optional[str] = None,
        app_root: Optional[str] = None,
        tls_key_path: Optional[str] = None,
        tls_cert_path: Optional[str] = None,
    ):
        _app_root: str = app_root if app_root else self._DEFAULT_APP_ROOT

        if not cwd:
            raise ValueError('cwd must be set')
        if not os.path.exists(cwd):
            raise ValueError(f'cwd does not exist: {cwd}')

        self._cwd = os.path.abspath(cwd)

        self._sqlite_db_path: str = os.path.abspath(os.path.join(self._cwd, "test", "tmp", "robot_cli_affirmation_store.db"))

        self._log_root: str = os.path.abspath(os.path.join(self._cwd, 'test', 'robot', 'log'))

        self._log_root: str = log_root if log_root else self._log_root

        self._affirmation_store_web_service = None

        self._web_server_app: str = f'{_app_root}/oauth2/token_srv'
        self._github_app: str = f'{_app_root}/github/app'
        self._gcp_app: str = f'{_app_root}/gcp/app'
        self._okta_app: str = f'{_app_root}/okta/app'
        self._static_auth_testing_app: str = f'{_app_root}/static_auth/app'
        self._aws_app: str = f'{_app_root}/aws/app'
        self._azure_app: str = f'{_app_root}/azure/app'
        self._digitalocean_app: str = f'{_app_root}/digitalocean/app'
        self._googleadmin_app: str = f'{_app_root}/googleadmin/app'
        self._k8s_app: str = f'{_app_root}/k8s/app'
        self._registry_app: str = f'{_app_root}/registry/app'
        self._sumologic_app: str = f'{_app_root}/sumologic/app'

        self._tls_key_path: str = tls_key_path if tls_key_path else self._DEFAULT_TLS_KEY_PATH
        self._tls_cert_path: str = tls_cert_path if tls_cert_path else self._DEFAULT_TLS_CERT_PATH
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
            f'--cert={self._tls_cert_path}',
            f'--key={self._tls_key_path}',
            stdout=os.path.abspath(os.path.join(self._log_root, f'token-client-credentials-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'token-client-credentials-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'github-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'github-server-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'gcp-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'gcp-server-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'okta-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'okta-server-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'aws-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'aws-server-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'static-auth-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'static-auth-server-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'google-admin-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'google-admin-server-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'k8s-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'k8s-server-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'registry-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'registry-server-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'azure-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'azure-server-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'sumologic-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'sumologic-server-{port}-stderr.txt')),
            cwd=self._cwd,
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
            stdout=os.path.abspath(os.path.join(self._log_root, f'digitalocean-server-{port}-stdout.txt')),
            stderr=os.path.abspath(os.path.join(self._log_root, f'digitalocean-server-{port}-stderr.txt')),
            cwd=self._cwd,
        )
    
    @keyword
    def start_all_webservers(self, port_dict: Optional[dict] = None) -> None:
        # if system has docker installed, use that to run mock servers
        if os.system('which docker >/dev/null 2>&1') == 0:
            ## inherits env vars from parent process so IS_DOCKER env var is passed along
            rv = os.system('docker compose -f docker-compose-testing.yml up -d --build --force-recreate')
            if rv != 0:
                raise RuntimeError('failed to start mock servers via docker compose')
            return


        _port_dict: dict = port_dict if port_dict else {}

        self.create_digitalocean_web_service(_port_dict.get('digitalocean', self._DEFAULT_MOCKSERVER_PORT_DIGITALOCEAN))
        self.create_sumologic_web_service(_port_dict.get('sumologic', self._DEFAULT_MOCKSERVER_PORT_SUMOLOGIC))
        self.create_registry_web_service(_port_dict.get('registry', self._DEFAULT_MOCKSERVER_PORT_REGISTRY))
        self.create_k8s_web_service(_port_dict.get('k8s', self._DEFAULT_MOCKSERVER_PORT_K8S))
        self.create_google_admin_web_service(_port_dict.get('googleadmin', self._DEFAULT_MOCKSERVER_PORT_GOOGLEADMIN))
        self.create_azure_web_service(_port_dict.get('azure', self._DEFAULT_MOCKSERVER_PORT_AZURE))
        self.create_aws_web_service(_port_dict.get('aws', self._DEFAULT_MOCKSERVER_PORT_AWS))
        self.create_static_auth_web_service(_port_dict.get('static_auth_testing', self._DEFAULT_MOCKSERVER_PORT_STACKQL_AUTH_TESTING))
        self.create_okta_web_service(_port_dict.get('okta', self._DEFAULT_MOCKSERVER_PORT_OKTA))
        self.create_gcp_web_service(_port_dict.get('gcp', self._DEFAULT_MOCKSERVER_PORT_GOOGLE))
        self.create_github_web_service(_port_dict.get('github', self._DEFAULT_MOCKSERVER_PORT_GITHUB))
        self.create_oauth2_client_credentials_web_service(_port_dict.get('oauth_client_credentials_token', self._DEFAULT_MOCKSERVER_PORT_OAUTH_CLIENT_CREDENTIALS_TOKEN))

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
