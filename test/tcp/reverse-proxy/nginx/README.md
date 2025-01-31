

## Runnning nginx as a pass through TCP proxy


### Automated testing scenario

This scenario requires `nginx` >= version `1.27.3` (and all of the `pip` requirements of this repository).

This is not ideal for local running; great for CI lifecycles.  If you do this locally, you **must** restore `/etc/hosts`.

We will need superuser privileges at times (via `sudo`) because we are editing `/etc/hosts` and using the protected port `443`.

Please run all commands from the root of the repository:

```bash
{
  echo '127.0.0.1   storage.googleapis.com'
} | sudo tee -a /etc/hosts

sudo nginx -c $(pwd)/test/tcp/reverse-proxy/nginx/tls-pass-through.conf

```

Then, run the robot tests that are suitable (as tags evolve, you may want to be more selective):

```bash

robot \
  --variable 'SUNDRY_CONFIG:{"registry_path": "test/registry"}' \
  --include tls_proxied \
  -d test/robot/reports \
  test/robot/functional

```

**Very important**; after this, restore `/etc/hosts` (BSD and GNU compatible, clunky expression):

```bash

sed '/storage.googleapis.com/d' /etc/hosts | sudo tee /etc/hosts

```
