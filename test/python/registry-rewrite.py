
import argparse
import os

from pathlib import Path

import shutil
import yaml

CURDIR :str = os.path.dirname(os.path.realpath(__file__))
TEST_ROOT_DIR :str = os.path.abspath(os.path.join(CURDIR, '..'))
REPOSITORY_ROOT_DIR :str = os.path.abspath(os.path.join(CURDIR, '../..'))

DEFAULT_SRC_DIR = os.path.join(TEST_ROOT_DIR, 'registry')
DEFAULT_DST_DIR = os.path.join(TEST_ROOT_DIR, 'registry-mocked')
DEFAULT_PORT = 1080



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
    '--port',
    type=int,
    default=DEFAULT_PORT,
    help='directory containing config and cache'
)

def main(args):
    os.chdir(args.srcdir)
    for r, dz, fz in os.walk('.'):
      for d in dz:
        Path(os.path.join(os.path.abspath(args.destdir), r, d)).mkdir(parents=True, exist_ok=True)
      for f in fz:
        if f.endswith('.yaml'):
          with open(os.path.join(r, f)) as fr:
            d = yaml.safe_load(fr)
          servs = d.get('servers', [])
          for srv in servs:
            srv['url'] = f'https://localhost:{args.port}/'
          d['servers'] = servs
          with open(os.path.join(os.path.abspath(args.destdir), r, f), 'w') as fw:
            yaml.dump(d, fw)
        else:
          shutil.copy(
            os.path.join(os.path.abspath(args.srcdir), r, f),
            os.path.join(os.path.abspath(args.destdir), r, f)
          )




if __name__ == '__main__':
    args = parser.parse_args()
    main(args)
