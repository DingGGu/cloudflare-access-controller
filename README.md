<p align="center">
<h2>Cloudflare Access Controller</h2>
[![Build Status](https://travis-ci.org/DingGGu/cloudflare-access-controller.svg?branch=master)](https://travis-ci.org/DingGGu/cloudflare-access-controller)
<h5>Support handling cloudflare access via kubernetes ingress annotations.</h5>
</p>

### List of annotations for configuration
- **access.cloudflare.com/zone-name**: 'ggu.la' # domain name
- **access.cloudflare.com/session-duration**: 30m, 6h, 12h, 24h, 168h, 730h
- access.cloudflare.com/application-sub-domain: 'subdomain'
- access.cloudflare.com/application-path: '/path-url'
- **access.cloudflare.com/policies**: "[]" # https://api.cloudflare.com/#access-policy-create-access-policy