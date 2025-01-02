
## Background

In this walkthrough, we go through the setup of a webserver using `stackql`.  This is useful in itself for development purposes, and we will build on it in more complex examples.

This walkthrough is not at all original; it is an amalgam of materials freely (and redundantly) available elsewehere.  It is heavily inspired by:

- [GCP documentation on running VMs with startup scripts](https://cloud.google.com/compute/docs/instances/startup-scripts/linux#rest).
- [GCP quickstart Apache on VM documentation](https://cloud.google.com/compute/docs/tutorials/basic-webserver-apache).
- [GCP quickstart Flask on VM documentation](https://cloud.google.com/docs/terraform/deploy-flask-web-server).
- [F5 Nginx install documentation](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-open-source/).


**NOTE** if your LAN / wifi has some firewall blocking connections to port 80, then this demonstration will not work.

## Setup

First, create a google service account key using the GCP Console, per [the GCP documentation](https://cloud.google.com/iam/docs/keys-create-delete).  Grant the service account at least requisite compute and firewall mutation privileges, per [the GCP documentation](https://cloud.google.com/iam/docs/create-service-agents#grant-roles);  corresponding to [this flask deployment example](https://cloud.google.com/docs/terraform/deploy-flask-web-server#permissions):


>  - `compute.instances.*`
>  - `compute.firewalls.*`

Then, do this in bash:

```bash setup stackql-shell credentials_path=cicd/keys/testing/google-rw-credentials.json app_root_path=./test/tmp/.create-google-vm-webserver.stackql

export GOOGLE_CREDENTIALS="$(cat <<credentials_path>>)";

stackql shell --approot=<<app_root_path>>
```

## Method

Do this in the `stackql` shell, replacing `<<project>>` with your GCP project name, '<<region>>', and `<<zone>>` as desired, eg: `australia-southeast1-a`:

```sql stackql-shell input required my_ephemeral_network_name=my-ephemeral-network-01 my_vm_name=my-ephemeral-vm-01 project=stackql-demo region=australia-southeast2 zone=australia-southeast2-a fw_name=ephemeral-http-01

registry pull google;

insert /*+ AWAIT */ into 
google.compute.networks (
  project,
  data__name,
  data__autoCreateSubnetworks
)
select
  '<<project>>',
  '<<my_ephemeral_network_name>>',
  true
;

insert /*+ AWAIT */ into 
google.compute.instances (
  project,
  zone,
  data__name,
  data__machineType,
  data__metadata,
  data__networkInterfaces,
  data__disks
)
select 
  '<<project>>',
  '<<zone>>',
  '<<my_vm_name>>',
  'zones/<<zone>>/machineTypes/n1-standard-1',
  '{
    "items": [
      {
        "key": "startup-script",
        "value": "#! /bin/bash\\nsudo apt-get update\\nsudo apt-get -y install apache2\\necho ''<!doctype html><html><body><h1>Hello from stackql auto-provisioned.</h1></body></html>'' | sudo tee /var/www/html/index.html"
      }
    ]
  }',
  '[
      {
        "stackType": "IPV4_ONLY",
        "accessConfigs": [
          {
            "name": "External NAT",
            "type": "ONE_TO_ONE_NAT",
            "networkTier": "PREMIUM"
          }
        ],
        "subnetwork": "projects/<<project>>/regions/<<region>>/subnetworks/<<my_ephemeral_network_name>>"
      }
    ]',
    '[
      {
        "autoDelete": true,
        "boot": true,
        "initializeParams": {
          "diskSizeGb": "10",
          "sourceImage": "https://compute.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/family/ubuntu-2004-lts"
        },
        "mode": "READ_WRITE",
        "type": "PERSISTENT"
      }
    ]'
;

insert /*+ AWAIT */ into 
google.compute.firewalls (
   project,
   data__name,
   data__network,
   data__allowed,
   data__direction,
   data__sourceRanges
)
select
  '<<project>>',
  '<<fw_name>>',
  'global/networks/<<my_ephemeral_network_name>>',
  '[
    {
      "IPProtocol": "tcp",
      "ports": [
        "80"
        
      ]
    }
  ]',
  'INGRESS',
  '[
    "0.0.0.0/0"
  ]'
;

```

```bash setup credentials_path=cicd/keys/testing/google-rw-credentials.json app_root_path=./test/tmp/.create-google-vm-webserver.stackql my_vm_name=my-ephemeral-vm-01 project=stackql-demo zone=australia-southeast2 zone=australia-southeast2-a

export GOOGLE_CREDENTIALS="$(cat <<credentials_path>>)";

publicIpAddress=$(stackql --approot=<<app_root_path>> exec "select json_extract(\"networkInterfaces\", '\$[0].accessConfigs[0].natIP') as public_ipv4_address from   google.compute.instances where project = '<<project>>' and zone = '<<zone>>' and instance = '<<my_vm_name>>';" -o json | jq -r '.[0].public_ipv4_address')

echo "publicIpAddress=${publicIpAddress}"
result=""
for i in $(seq 1 20); do
  sleep 5;
  result="$(curl http://${publicIpAddress} | grep 'auto-provisioned')";
  if [ "${result}" != "" ]; then
    break
  fi
done

echo "${result}";

```

## Result


You will see exactly this in the output:

```html expectation stdout-contains-all
<!doctype html><html><body><h1>Hello from stackql auto-provisioned.</h1></body></html>
```

## Cleanup

```bash teardown best-effort app_root_path=./test/tmp/.create-google-vm-webserver.stackql credentials_path=cicd/keys/testing/google-rw-credentials.json my_ephemeral_network_name=my-ephemeral-network-01 my_vm_name=my-ephemeral-vm-01 project=stackql-demo region=australia-southeast2 zone=australia-southeast2-a fw_name=ephemeral-http-01

echo "begin teardown";

export GOOGLE_CREDENTIALS="$(cat <<credentials_path>>)";

stackql --approot=<<app_root_path>> exec "delete /*+ AWAIT */ from google.compute.instances where project = '<<project>>' and zone = '<<zone>>' and instance = '<<my_vm_name>>';"

stackql --approot=<<app_root_path>> exec "delete /*+ AWAIT */ from google.compute.firewalls where project = '<<project>>' and firewall= '<<fw_name>>';"

stackql --approot=<<app_root_path>> exec "delete /*+ AWAIT */ from google.compute.networks where project = '<<project>>' and network = '<<my_ephemeral_network_name>>';"

rm -rf <<app_root_path>> ;

echo "conclude teardown";

```
