
import argparse
import os

from pathlib import Path

import urllib.parse

import shutil
import yaml

DEFAULT_PORT = 1070
GOOGLE_DEFAULT_PORT = 1080
GOOGLEADMIN_DEFAULT_PORT = 1098
OKTA_DEFAULT_PORT = 1090
AWS_DEFAULT_PORT = 1091
K8S_DEFAULT_PORT = 1092
GITHUB_DEFAULT_PORT = 1093
AZURE_DEFAULT_PORT = 1095
SUMOLOGIC_DEFAULT_PORT = 1096
DIGITALOCEAN_DEFAULT_PORT = 1097
STACKQL_TEST_DEFAULT_PORT = 1099
STACKQL_AUTH_TESTING_DEFAULT_PORT = 1170



parser = argparse.ArgumentParser(description='Process some test config.')
parser.add_argument(
    '--srcdir', 
    type=str,
    required=True,
    help='directory containing executable'
)
parser.add_argument(
    '--destdir',
    type=str,
    required=True,
    help='directory containing config and cache'
)
parser.add_argument(
    '--replacement-host',
    type=str,
    default='localhost',
    help='host name to overwrite in docs'
)
parser.add_argument(
    '--default-port',
    type=int,
    default=DEFAULT_PORT,
    help='fallback port for default mock service'
)
parser.add_argument(
    '--google-port',
    type=int,
    default=GOOGLE_DEFAULT_PORT,
    help='port for google mock service'
)
parser.add_argument(
    '--googleadmin-port',
    type=int,
    default=GOOGLEADMIN_DEFAULT_PORT,
    help='port for google mock service'
)
parser.add_argument(
    '--stackqltest-port',
    type=int,
    default=STACKQL_TEST_DEFAULT_PORT,
    help='port for stackql test service'
)
parser.add_argument(
    '--okta-port',
    type=int,
    default=OKTA_DEFAULT_PORT,
    help='port for okta mock service'
)
parser.add_argument(
    '--aws-port',
    type=int,
    default=AWS_DEFAULT_PORT,
    help='port for aws mock service'
)
parser.add_argument(
    '--k8s-port',
    type=int,
    default=K8S_DEFAULT_PORT,
    help='port for k8s mock service'
)
parser.add_argument(
    '--github-port',
    type=int,
    default=GITHUB_DEFAULT_PORT,
    help='port for github mock service'
)
parser.add_argument(
    '--azure-port',
    type=int,
    default=AZURE_DEFAULT_PORT,
    help='port for azure mock service'
)
parser.add_argument(
    '--sumologic-port',
    type=int,
    default=SUMOLOGIC_DEFAULT_PORT,
    help='port for sumologic mock service'
)
parser.add_argument(
    '--digitalocean-port',
    type=int,
    default=DIGITALOCEAN_DEFAULT_PORT,
    help='port for digitalocean mock service'
)
parser.add_argument(
    '--stackql-auth-testing-port',
    type=int,
    default=STACKQL_AUTH_TESTING_DEFAULT_PORT,
    help='port for stackql auth test mock service'
)

class ProviderArgs:

  def __init__(self, name :str, srcdir :str, destdir :str, port :int, replacement_host :str='localhost'):
    self.name            = name
    self.srcdir          = srcdir
    self.destdir         = destdir
    self.port            = port
    self.replacement_host = replacement_host

  def isServerRewriteRequired(self) -> bool:
    return self.name != 'k8s' 
  
def _replace_token_url(url :str, replacement_host :str) -> str:
  parsed = urllib.parse.urlparse(url)
  replaced = parsed._replace(netloc="{}:{}".format(replacement_host, parsed.port))
  return urllib.parse.urlunparse(replaced)

def _replace_server_url(url :str, replacement_host :str, replacement_port: int) -> str:
  parsed = urllib.parse.urlparse(url)
  replaced = parsed._replace(netloc="{}:{}".format(replacement_host, replacement_port))
  return urllib.parse.urlunparse(replaced)

def rewrite_provider(args :ProviderArgs):
    os.chdir(args.srcdir)
    for r, dz, fz in os.walk('.'):
      for d in dz:
        Path(os.path.join(os.path.abspath(args.destdir), r, d)).mkdir(parents=True, exist_ok=True)
      for f in fz:
        if f.endswith('.yaml'):
          with open(os.path.join(r, f), encoding="utf8") as fr:
            d = yaml.safe_load(fr)
          servs = d.get('servers', [])
          if args.isServerRewriteRequired():
            for srv in servs:
              srv['url'] = _replace_server_url(srv['url'], args.replacement_host, args.port) 
          d['servers'] = servs
          token_url = d.get('config', {}).get('auth', {}).get('token_url')
          if args.isServerRewriteRequired() and token_url:
            d['config']['auth']['token_url'] = _replace_token_url(token_url, args.replacement_host)
          for path, path_item in d.get('paths', {}).items():
            path_item_servers = path_item.get('servers', [])
            if args.isServerRewriteRequired():
              for srv in path_item_servers:
                srv['url'] = _replace_server_url(srv['url'], args.replacement_host, args.port) 
                path_item_servers
            for k in ('get', 'put', 'post', 'delete', 'head'):
              operation = path_item.get(k)
              if operation:
                operation_servers = operation.get('servers', [])
                if args.isServerRewriteRequired():
                  for srv in operation_servers:
                    srv['url'] = _replace_server_url(srv['url'], args.replacement_host, args.port) 
          graphql_url = d.get('paths', {}).get('/graphql', {}).get('post', {}).get('x-stackQL-graphQL', {}).get('url')
          if graphql_url:
            d['paths']['/graphql']['post']['x-stackQL-graphQL']['url'] = f'{_replace_server_url(graphql_url, args.replacement_host, args.port)}/graphql'
          with open(os.path.join(os.path.abspath(args.destdir), r, f), 'w') as fw:
            yaml.dump(d, fw)
        else:
          shutil.copy(
            os.path.join(os.path.abspath(args.srcdir), r, f),
            os.path.join(os.path.abspath(args.destdir), r, f)
          )


class ProviderCfgMapping:

  def __init__(self, processed_args :argparse.Namespace) -> None:
    self._provider_lookup :dict = {
      "aws": {
        "port": processed_args.aws_port
      },
      "azure": {
        "port": processed_args.azure_port
      },
      "github": {
        "port": processed_args.github_port
      },
      "googleadmin": {
        "port": processed_args.googleadmin_port
      },
      "googleapis.com": {
        "port": processed_args.google_port
      },
      "okta": {
        "port": processed_args.okta_port
      },
      "k8s": {
        "port": processed_args.k8s_port
      },
      "sumologic": {
        "port": processed_args.sumologic_port
      },
      "digitalocean": {
        "port": processed_args.digitalocean_port
      },
      "stackql_test": {
        "port": processed_args.stackqltest_port
      },
      "stackql_auth_testing": {
        "port": processed_args.stackql_auth_testing_port
      },
      "stackql_oauth2_testing": {
        "port": processed_args.stackql_auth_testing_port # shared port acceptable coz auth server decooupled for outh2
      },
      "__default__": {
        "port": processed_args.default_port
      }
    }

  def get_port(self, provider_name :str) -> int:
    return self._provider_lookup.get(provider_name, self._provider_lookup.get("__default__")).get("port", DEFAULT_PORT)


def rewrite_registry():
    args = parser.parse_args()
    ppm = ProviderCfgMapping(args)
    provider_dirs = [f for f in os.scandir(args.srcdir) if f.is_dir()]
    for prov_dir in provider_dirs:
      prov_args = ProviderArgs(
        prov_dir.name,
        prov_dir.path,
        os.path.join(args.destdir, prov_dir.name),
        ppm.get_port(prov_dir.name),
        replacement_host=args.replacement_host,
      ) 
      print(f'{prov_args.srcdir}, {prov_args.destdir}, {prov_args.port}')
      rewrite_provider(prov_args)

if __name__ == '__main__':
    rewrite_registry()
