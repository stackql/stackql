// variables
local name = 'env-var-input-demo';
local project = std.extVar("project");
local region = std.extVar("region");
local self_link_base = 'https://compute.googleapis.com/compute/v1/projects/' + project + '/';
local self_link_global = self_link_base + 'global/';
local nw_name = name + '-vpc';
local sourceImage = 'https://compute.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/family/ubuntu-2004-lts';
local diskSizeGb = '10';

{
  // global config
  global: {
    project: project,
    region: region
  },
  // network
  network: {
    autoCreateSubnetworks: false,
    name: name + '-vpc',
    routingConfig: {routingMode: 'REGIONAL'}
  },
  // subnet
  subnetwork: {
    ipCidrRange: '10.240.0.0/24',
    name: name + '-subnet',
    network: self_link_global + 'networks/' + nw_name,
    privateIpGoogleAccess: false
  }
}