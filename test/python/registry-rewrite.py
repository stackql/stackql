
import argparse
import os

from pathlib import Path

import shutil
import yaml

CURDIR :str = os.path.dirname(os.path.realpath(__file__))
TEST_ROOT_DIR :str = os.path.abspath(os.path.join(CURDIR, '..'))
REPOSITORY_ROOT_DIR :str = os.path.abspath(os.path.join(CURDIR, '../..'))

DEFAULT_SRC_DIR = os.path.join(TEST_ROOT_DIR, 'registry', 'src')
DEFAULT_DST_DIR = os.path.join(TEST_ROOT_DIR, 'registry-mocked', 'src')
DEFAULT_PORT = 1070
GOOGLE_DEFAULT_PORT = 1080
OKTA_DEFAULT_PORT = 1090
AWS_DEFAULT_PORT = 1091
K8S_DEFAULT_PORT = 1092
GITHUB_DEFAULT_PORT = 1093



parser = argparse.ArgumentParser(description='Process some test config.')
parser.add_argument(
    '--srcdir', 
    type=str,
    default=DEFAULT_SRC_DIR,
    help='directory containing executable'
)
parser.add_argument(
    '--destdir',
    type=str,
    default=DEFAULT_DST_DIR,
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

class ProviderArgs:

  def __init__(self, name :str, srcdir :str, destdir :str, port :int, replacement_host :str='localhost'):
    self.name            = name
    self.srcdir          = srcdir
    self.destdir         = destdir
    self.port            = port
    self.replacement_host = replacement_host

  def isServerRewriteRequired(self) -> bool:
    return self.name != 'k8s' 

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
              srv['url'] = f'https://{args.replacement_host}:{args.port}/'
          d['servers'] = servs
          graphql_url = d.get('paths', {}).get('/graphql', {}).get('post', {}).get('x-stackQL-graphQL', {}).get('url')
          if graphql_url:
            d['paths']['/graphql']['post']['x-stackQL-graphQL']['url'] = f'https://{args.replacement_host}:{args.port}/graphql'
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
      "github": {
        "port": processed_args.github_port
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
      "__default__": {
        "port": processed_args.default_port
      }
    }

  def get_port(self, provider_name :str) -> int:
    return self._provider_lookup.get(provider_name, self._provider_lookup.get("__default__")).get("port", DEFAULT_PORT)


if __name__ == '__main__':
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
