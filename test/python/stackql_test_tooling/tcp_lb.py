import argparse

from typing import List, Tuple, Iterable, Optional

_GOOGLE_DEFAULT_PORT = 1080
_AWS_DEFAULT_PORT = 1091

_DEFAULT_AWS_GLOBAL_SERVICES: Tuple[str] = (
    'iam',
    'route53',
)

_DEFAULT_AWS_REGIONAL_SERVICES: Tuple[str] = (
    "ce", # costexplorer
    "cloudcontrolapi", # cloudcontrol
    'cloudhsmv2', # cloudhsm
    "ec2",
    'logs', # cloudwatch
    "s3",
    "transfer",
)

_DEFAULT_AWS_REGIONS: Tuple[str] = (
    "us-east-1",
    "us-east-2",
    "us-west-1",
    "us-west-2",
    "ap-south-1",
    "ap-northeast-1",
    "ap-northeast-2",
    "ap-southeast-1",
    "ap-southeast-2",
    "ap-northeast-3",
    "ca-central-1",
    "eu-central-1",
    "eu-west-1",
    "eu-west-2",
    "eu-west-3",
    "eu-north-1",
    "sa-east-1",
)

_DEFAULT_GCP_SERVICES: Tuple[str] = (
    'compute',
    'storage',
)

class _LB(object):

    def __init__(self, lb_host: str, lb_port, backend_host: str, backend_port: int) -> None:
        self._lb_host = lb_host
        self._lb_port = lb_port
        self._backend_host = backend_host
        self._backend_port = backend_port
    
    def get_lb_host(self) -> str:
        return self._lb_host
    
    def get_lb_port(self) -> int:
        return self._lb_port
    
    def get_backend_host(self) -> str:
        return self._backend_host
    
    def get_backend_port(self) -> int:
        return self._backend_port

class _HostsGenerator(object):

    def __init__(
            self, 
            aws_global_services: Optional[Iterable[str]] = None, 
            aws_regional_services: Optional[Iterable[str]] = None, 
            aws_regions: Optional[Iterable[str]] = None, 
            google_services: Optional[Iterable[str]] = None
        ) -> None:
        self._aws_global_services = aws_global_services if aws_global_services is not None else _DEFAULT_AWS_GLOBAL_SERVICES
        self._aws_regional_services = aws_regional_services if aws_regional_services is not None else _DEFAULT_AWS_REGIONAL_SERVICES
        self._aws_regions = aws_regions if aws_regions is not None else _DEFAULT_AWS_REGIONS
        self._google_services = google_services if google_services is not None else _DEFAULT_GCP_SERVICES

    @staticmethod
    def _generate_aws_hosts(aws_regional_services: Iterable[str], aws_regions: Iterable[str], aws_global_services: Iterable[str]) -> Iterable[_LB]:
        for service in aws_regional_services:
            for region in aws_regions:
                yield _LB(f'{service}.{region}.amazonaws.com', 443, '127.0.0.1', _AWS_DEFAULT_PORT)
        for service in aws_global_services:
            yield _LB(f'{service}.amazonaws.com', 443, '127.0.0.1', _AWS_DEFAULT_PORT)

    @staticmethod
    def _generate_gcp_hosts(google_services: Iterable[str]) -> Iterable[_LB]:
        for service in google_services:
            yield _LB(f'{service}.googleapis.com', 443, '127.0.0.1', _GOOGLE_DEFAULT_PORT)
        
    def generate_all_load_balancers(self) -> Iterable[_LB]:
        yield from self._generate_aws_hosts(self._aws_regional_services, self._aws_regions, self._aws_global_services)
        yield from self._generate_gcp_hosts(self._google_services)


class _NginxConfigGenerator(object):

    def __init__(self, n_indent: int = 2) -> None:
        self._n_indent = n_indent

    @staticmethod
    def _generate_backends(hosts: Iterable[_LB], indent: int) -> Iterable[str]:
        """
        Generate the nginx config.
        """
        return [
            " " * indent + f'{host.get_lb_host()}    {host.get_backend_host()}:{host.get_backend_port()};'
            for host in hosts
        ]
    
    def _generate_file_content(self, hosts: Iterable[_LB]) -> Tuple[str]:
        """
        Generate the file content.
        """
        lines : Tuple[str] = (
            'worker_processes  1;',
            '',
            '#error_log  logs/error.log;',
            '#error_log  logs/error.log  notice;',
            '#error_log  logs/error.log  info;',
            '',
            '#pid        logs/nginx.pid;',
            '',
            '',
            'events {',
            f'{" " * self._n_indent}worker_connections  1024;',
            '}',
            '',
            'stream {',
            '',
            f'{" " * self._n_indent}map $ssl_preread_server_name $targetBackend {{',
            *self._generate_backends(hosts, self._n_indent * 2),
            f'{" " * self._n_indent}}}',
            '',
            f'{" " * self._n_indent}server {{',
            f'{" " * self._n_indent}listen 443;',
            f'{" " * self._n_indent}',
            f'{" " * self._n_indent}proxy_connect_timeout 1s;',
            f'{" " * self._n_indent}proxy_timeout 3s;',
            f'{" " * self._n_indent}resolver 1.1.1.1;',
            f'{" " * self._n_indent}',
            f'{" " * self._n_indent}proxy_pass $targetBackend;',
            f'{" " * self._n_indent}ssl_preread on;',
            f'{" " * self._n_indent}}}',
            '}',
        )
        return lines

    def generate_file_content(self, hosts: Iterable[_LB]) -> Tuple[str]:
        """
        Generate the file content.
        """
        return self._generate_file_content(hosts)


class _HostsFileEntriesGenerator(object):

    def __init__(self, hosts: Iterable[_LB]) -> None:
        self._hosts = hosts

    def generate_entries(self) -> Iterable[str]:
        """
        Generate the entries.
        """
        return tuple(
            f'{host.get_backend_host()}    {host.get_lb_host()}'
            for host in self._hosts
        )


def _parse_args() -> argparse.Namespace:
    """
    Parse the arguments.
    """
    parser = argparse.ArgumentParser(description='Create a token.')
    parser.add_argument('--generate-nginx-lb', help='Opt-in nginx config generation', action=argparse.BooleanOptionalAction)
    parser.add_argument('--generate-hosts-entries', help='Opt-in hosts files entries generation', action=argparse.BooleanOptionalAction)
    # parser.add_argument('--header', type=str, help='The header.')
    return parser.parse_args()

def generate_lb_config():
    args = _parse_args()
    host_gen = _HostsGenerator()
    all_hosts = [lb for lb in host_gen.generate_all_load_balancers()]
    nginx_cfg_gen = _NginxConfigGenerator()
    if args.generate_nginx_lb:
        for l in nginx_cfg_gen.generate_file_content(all_hosts):
            print(l)
        return
    if args.generate_hosts_entries:
        hosts_gen = _HostsFileEntriesGenerator(all_hosts)
        for l in hosts_gen.generate_entries():
            print(l)
        return

if __name__ == '__main__':
    generate_lb_config()
