# Cloudflare Access Controller
[![Build Status](https://travis-ci.org/DingGGu/cloudflare-access-controller.svg?branch=master)](https://travis-ci.org/DingGGu/cloudflare-access-controller)

Cloudflare Access Controller synchronizes Kubernetes Ingress with [Cloudflare Access](https://www.cloudflare.com/products/cloudflare-access/)

### Deploy
```bash
kubectl apply -f deploy/cloudflare-access-controller.yaml
```

### Configuration guide
Execute with the following command:
```bash
./cloudflare-access-controller \
-z cloudflare.zone.name \
-c identifier.cluster.name \
-i interval.time
```

#### Ingress Annotations
```yaml
annotations:
  access.cloudflare.com/zone-name: 'ggu.la' # (required) domain name
  access.cloudflare.com/session-duration: 30m, 6h, 12h, 24h, 168h, 730h # (required) domain name
  access.cloudflare.com/application-sub-domain: 'subdomain'
  access.cloudflare.com/application-path: '/path-url'
  access.cloudflare.com/policies: "[]" # https://api.cloudflare.com/#access-policy-create-access-policy
```

#### Policy Examples
- Allow login account email ends with ggu.la and mah.ye and IP address require 123.123.123.123/32 
```json
[{"decision":"allow","include":[{"email_domain":{"domain":"ggu.la"}},{"email_domain":{"domain":"mah.ye"}}],"require":[{"ip":{"ip":"123.123.123.123/32"}}]}]
```
- Bypass IP Address 123.123.123.123/32 and Denied IP Address 192.168.0.1/32
```json
[{"decision":"bypass","require":[{"ip":{"ip":"123.123.123.123/32"}}]},{"decision":"deny","require":[{"ip":{"ip":"192.168.0.1/32"}}]}]
``` 
- More example: https://developers.cloudflare.com/access/setting-up-access/configuring-access-policies/

### Other Tips
Cloudflare Access must be proxied (a.k.a orange cloud enabled). ExternalDNS makes it easy to manage Cloudflare's DNS with Kubernetes. It is strongly recommend using it with that.