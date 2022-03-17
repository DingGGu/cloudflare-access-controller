![Deploy to docker hub](https://github.com/DingGGu/cloudflare-access-controller/workflows/Deploy%20to%20docker%20hub/badge.svg)

# Cloudflare Access Controller

Cloudflare Access Controller synchronizes Kubernetes Ingress
with [Cloudflare Access](https://www.cloudflare.com/products/cloudflare-access/)

### Prerequisites

| Kubernetes Version | Cloudflare Access Controller Version |
|--------------------|--------------------------------------|
| > = 1.22            | > = 2.1.0                             |
| <= 1.21            | 2.0.0                                |

### Deploy

```bash
kubectl apply -f deploy/cloudflare-access-controller.yaml
```

Access Policy is created with the name `cac-policy-{#number}`. Policy without start `cac-policy-` is ignored, so it can
be configured by adding or changing it directly in your Cloudflare Dashboard.

### Configuration guide

Image is available
here: [ghcr.io/dingggu/cloudflare-access-controller:latest](https://github.com/users/DingGGu/packages/container/package/cloudflare-access-controller)

Execute with the following command:

```bash
./cloudflare-access-controller \
-z cloudflare.zone.name \
-c identifier.cluster.name
```

or figure out with

```
./cloudflare-access-controller -h
```

#### Ingress Annotations
```yaml
annotations:
  access.cloudflare.com/application-sub-domain: 'subdomain' # required, if set '', will applied domain
  access.cloudflare.com/application-path: '/path-url' # if not set, default '/'
  access.cloudflare.com/session-duration: 30m, 6h, 12h, 24h, 168h, 730h # if not set, default 24h 
  access.cloudflare.com/policies: |
    "[]"
  # https://api.cloudflare.com/#access-policy-create-access-policy
```

#### Policy Examples
- Allow login account email ends with ggu.la and mah.ye and IP address require 123.123.123.123/32 
```json
[{"decision":"allow","include":[{"email_domain":{"domain":"ggu.la"}},{"email_domain":{"domain":"google.com"}}],"require":[{"ip":{"ip":"123.123.123.123/32"}}]}]
```
- Bypass IP Address 123.123.123.123/32 and Denied IP Address 192.168.0.1/32
```json
[{"decision":"bypass","require":[{"ip":{"ip":"123.123.123.123/32"}}]},{"decision":"deny","require":[{"ip":{"ip":"192.168.0.1/32"}}]}]
``` 
- More example: https://developers.cloudflare.com/access/setting-up-access/configuring-access-policies/

### Other Tips
Cloudflare is recommended, as it is more secure when used with [Argo tunnels](https://developers.cloudflare.com/argo-tunnel/reference/kubernetes/).

If not use with Argo tunnel, Access must be proxied (a.k.a orange cloud enabled). [ExternalDNS](https://github.com/kubernetes-sigs/external-dns) makes it easy to manage Cloudflare's DNS with Kubernetes. It is strongly recommend using it with that.