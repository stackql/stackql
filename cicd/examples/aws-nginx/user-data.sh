#!/bin/bash
set -euxo pipefail
dnf install -y nginx
cat > /etc/nginx/nginx.conf <<'NGINX'
user nginx;
worker_processes auto;
pid /run/nginx.pid;
error_log /var/log/nginx/error.log;
events { worker_connections 1024; }
http {
 include /etc/nginx/mime.types;
 default_type application/octet-stream;
 access_log /var/log/nginx/access.log;
 server {
  listen 80 default_server;
  server_name _;
  root /usr/share/nginx/html;
  add_header X-Omnisdk-Deployment "__DEPLOYMENT__" always;
  location / { try_files $uri $uri/ =404; }
  location = /healthz { default_type text/plain; return 200 "omnisdk __DEPLOYMENT__ ready\n"; }
 }
}
NGINX
cat > /usr/share/nginx/html/index.html <<'HTML'
<!doctype html><html><head><title>omnisdk nginx</title></head><body><h1>omnisdk __DEPLOYMENT__</h1><p>nginx, provisioned through merged EC2 documents.</p></body></html>
HTML
nginx -t
systemctl enable --now nginx
