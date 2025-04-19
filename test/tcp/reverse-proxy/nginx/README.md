

## Runnning nginx as a pass through TCP proxy

These scenarios require `python` >= `3.11`, `nginx` >= `1.27.3` (and all of the `pip` requirements of this repository).

This is not ideal for local running; great for CI lifecycles.  If you do this locally, you **must** restore `/etc/hosts`.  
Rough and ready `/etc/hosts` restoration instructions are supplied, but, honestly, it is better if you take a prior backup and 
clobber with same when done.

We will need superuser privileges at times (via `sudo`) because we are editing `/etc/hosts` and using the protected port `443`.

### Automated testing scenario


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

### More fulsome testing

Please run all commands from the root of the repository:

```bash

python test/python/stackql_test_tooling/tcp_lb.py --generate-hosts-entries | sudo tee -a /etc/hosts

python test/python/stackql_test_tooling/tcp_lb.py --generate-nginx-lb > test/tcp/reverse-proxy/nginx/dynamic-sni-proxy.conf

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

sed '/googleapis/d' /etc/hosts | sudo tee /etc/hosts

sed '/aws/d' /etc/hosts | sudo tee /etc/hosts

```
