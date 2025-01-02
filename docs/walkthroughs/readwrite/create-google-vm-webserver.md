
## Background

In this walkthrough, we go through the setup of a webserver using `stackql`.  This is useful in itself for development purposes, and we will build on it in more complex examples.

This walkthrough is not at all original; it is an amalgam of materials freely (and redundantly) available elsewehere.  It is heavily inspired by:

- [GCP documentation on running VMs with startup scripts](https://cloud.google.com/compute/docs/instances/startup-scripts/linux#rest).
- [GCP quickstart Apache on VM documentation](https://cloud.google.com/compute/docs/tutorials/basic-webserver-apache).
- [GCP quickstart Flask on VM documentation](https://cloud.google.com/docs/terraform/deploy-flask-web-server).
- [F5 Nginx install documentation](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-open-source/).

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

```sql stackql-shell input required my_ephemeral_network_name=my-ephemeral-network-01 my_vm_name=my-ephemeral-vm-01 project=stackql-demo region=australia-southeast2 zone=australia-southeast2-a 

registry pull google;

insert into 
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

insert into 
google.compute.instances (
  project,
  zone,
  data__name,
  data__metadata,
  data__networkInterfaces
)
select 
  '<<project>>',
  '<<zone>>',
  '<<my_vm_name>>',
  true,
  '{
    "items": [
      {
        "key": "startup-script",
        "value": "#! /bin/bash\nsudo apt-get update\nsudo apt-get -y install apache2\necho '<!doctype html><html><body><h1>Hello from stackql auto-provisioned.</h1></body></html>' | sudo tee /var/www/html/index.html"
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
    ]'
;

```

```bash expect-stdoout-contains=auto-provisioned app_root_path=./test/tmp/.create-google-vm-webserver.stackql my_vm_name=my-ephemeral-vm-01 project=stackql-demo zone=australia-southeast2 zone=australia-southeast2-a

export GOOGLE_CREDENTIALS="$(cat <<credentials_path>>)";

publicIpAddress=$(stackql --approot=<<app_root_path>> exec "select json_extract(\"networkInterfaces\", '\$[0].accessConfigs[0].natIP') as public_ipv4_address from   google.compute.instances where project = '<<project>>' and zone = '<<zone>>' and instance = '<<my_vm_name>>';" -o json | jq -r '.[0].public_ipv4_address')

curl http://${publicIpAddress} | grep 'auto-provisioned'
```

## Result


You will see exactly this in the output:

```html expectation stdout-contains-all
<!doctype html><html><body><h1>Hello from stackql auto-provisioned.</h1></body></html>
```

## Cleanup

```bash teardown best-effort app_root_path=./test/tmp/.create-google-vm-webserver.stackql my_ephemeral_network_name=my-ephemeral-network-01 my_vm_name=my-ephemeral-vm-01 project=stackql-demo region=australia-southeast2 zone=australia-southeast2-a 

stackql --approot=<<app_root_path>> exec "delete from google.compute.instances where project = '<<project>>' and zone = '<<zone>>' and instance = '<<my_vm_name>>';"

stackql --approot=<<app_root_path>> exec "delete from google.compute.networks where project = '<<project>>' and zone = '<<zone>>' and network = '<<my_ephemeral_network_name>>';"

rm -rf <<app_root_path>>

```


### Working

#### Network

`https://compute.googleapis.com/compute/v1/projects/{project}/global/networks`.


```json
{
  "name": "my-shortlived-nw-01",
  "autoCreateSubnetworks": true
  
}
```

#### Firewall

`POST https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls`.

```json
{
  "name": "ephemeral-http-01",
  "network": "global/networks/my-shortlived-nw-01",
  "allowed": [
    {
      "IPProtocol": "tcp",
      "ports": [
        "80"
        
      ]
    }
    
  ],
  "direction": "INGRESS",
  "sourceRanges": [
    "0.0.0.0/0"
    
  ]
  
}

```

#### VM

```json
{
  "name": "my-ephemeral-vm-02",
  "machineType": "zones/australia-southeast2-a/machineTypes/n1-standard-1",
  "disks": [
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
    
  ],
  "networkInterfaces": [
    {
      "stackType": "IPV4_ONLY",
      "accessConfigs": [
        {
          "name": "External NAT",
          "type": "ONE_TO_ONE_NAT",
          "networkTier": "PREMIUM"
          
        }
        
      ],
      "subnetwork": "projects/ryuki-it-sandbox-01/regions/australia-southeast2/subnetworks/my-shortlived-nw-01"
      
    }
    
  ],
  "metadata": {
    "items": [
      {
        "key": "startup-script",
        "value": "#! /bin/bash\nsudo apt-get update\nsudo apt-get -y install apache2\necho '<!doctype html><html><body><h1>Hello from stackql auto-provisioned.</h1></body></html>' | sudo tee /var/www/html/index.html"
      }
      
    ]
    
  }
  
}
```

---

An attempt:

```json
{
  "name": "my-silly-vm-02",
  "networkInterfaces": [
    {
      "subnetwork": "projects/<<project>>/regions/australia-southeast2/subnetworks/<<my_ephemeral_network_name>>"
      
    }
    
  ],
  "metadata": {
    "items": [
      {
        "key": "startup-script",
        "value": "#! /bin/bash\napt update\napt -y install apache2\ncat <<EOF > /var/www/html/index.html\n<html><body><p>Linux startup script added directly.</p></body></html>\nEOF"
      }
      
    ]
    
  },
  "machineType": "zones/australia-southeast2-a/machineTypes/n1-standard-1",
  "disks": [
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
    
  ]
  
}
```

## Result


You will see something very much like this included in the output, presuming you have one VM (if you have zero, only the headers should appper, more VMs means more rows):

```sql stackql stdout expectation stdout-table-contains-data
|--------------------------------------------------|---------------------|
|                       name                       |         id          |
|--------------------------------------------------|---------------------|
| any-compute-cluster-1-default-abcd-00000001-0001 | 1000000000000000001 |
|--------------------------------------------------|---------------------|
```

<!---  EXPECTATION
google\ provider,\ version\ 'v24.11.00274'\ successfully\ installed
goodbye
-->

<x-expectation style="display: none;">
<stdout-contains-nonempty-table></stdout-contains-nonempty-table>
</x-expectation>

## Cleanup

```bash teardown best-effort app_root_path=./test/tmp/.create-google-vm-webserver-qs.stackql

rm -rf <<app_root_path>>

```