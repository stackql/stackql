#!/usr/bin/env python3
"""Prepare local inputs and a document merge request. Resolves an AMI read-only; never provisions cloud resources."""
import argparse
import base64
import json
import ipaddress
from pathlib import Path
import re
import subprocess
import uuid
from image_inputs import resolve_image

p=argparse.ArgumentParser(description=__doc__)
p.add_argument('--case',type=Path,required=True)
p.add_argument('--region', required=True)
p.add_argument('--image-id', help='Concrete AMI ID; otherwise resolve AL2023 through AWS SSM once and persist it')
p.add_argument('--zone-a',required=True)
p.add_argument('--zone-b',required=True)
p.add_argument('--ssh-cidr',required=True)
p.add_argument('--http-cidr',default='0.0.0.0/0')
p.add_argument('--prefix',default='omni-web')
p.add_argument('--instance-type',default='t3.micro')
p.add_argument('--public-key',type=Path)
p.add_argument('--poll-interval',default='2s')
p.add_argument('--poll-attempts',type=int,default=120)
a=p.parse_args()
if not re.fullmatch(r'[A-Za-z0-9-]{1,40}',a.prefix):p.error('invalid prefix')
for cidr in (a.ssh_cidr,a.http_cidr):
    if ipaddress.ip_network(cidr).version!=4:p.error('IPv4 CIDRs required')
if a.poll_attempts<1:p.error('poll attempts must be positive')
a.case=a.case.resolve();a.case.mkdir(mode=0o700,parents=True,exist_ok=True)
if (a.case/'inputs.json').exists():p.error('case already prepared; use its inputs and ledger, or a new directory')
try:
    a.image_id = resolve_image(a.image_id, a.region)
except (ValueError, OSError, subprocess.SubprocessError) as error:
    p.error('cannot pin AMI: ' + str(error))

key=a.public_key
if key is None:
    private=a.case/'id_ed25519'
    if private.exists():p.error('refusing to overwrite local SSH key')
    subprocess.run(['ssh-keygen','-q','-t','ed25519','-N','','-C',a.prefix,'-f',str(private)],check=True)
    key=private.with_suffix('.pub')
public=key.read_text().strip()
if not public.startswith('ssh-'):p.error('OpenSSH public key required')
base=Path(__file__).resolve().parent
script=(base/'user-data.sh').read_text().replace('__DEPLOYMENT__',a.prefix)
inputs={'image_id':a.image_id,'instance_type':a.instance_type,'zone_a':a.zone_a,'zone_b':a.zone_b,'ssh_cidr':a.ssh_cidr,'http_cidr':a.http_cidr,'key_name':a.prefix+'-ssh','group_name':a.prefix+'-web','client_token':uuid.uuid4().hex,'public_key_material':base64.b64encode(public.encode()).decode(),'user_data':base64.b64encode(script.encode()).decode()}
# Tags are user inputs to the static deployment specification.
for resource in json.loads((base/'resources.json').read_text())['resources']:
    inputs['tag_'+resource['key']]=a.prefix+'-'+resource['key']
patch=json.loads((base/'ec2.merge.json').read_text())
for resource in patch['components']['x-stackQL-resources'].values():
    for method in resource['methods'].values():
        waiter=method.get('x-stackQL-beta',{}).get('await')
        if waiter:waiter.update(interval=a.poll_interval,maxAttempts=a.poll_attempts)
for name,data in {'inputs.json':inputs,'merge.json':{'patches':[{'service':'stackql_unstable_aws.ec2','merge':patch}],'doc_cache':{'dir':str(a.case/'registry-cache')}}}.items():
    path=a.case/name;path.write_text(json.dumps(data,indent=2)+'\n');path.chmod(0o600)
(a.case/'user-data.sh').write_text(script)
