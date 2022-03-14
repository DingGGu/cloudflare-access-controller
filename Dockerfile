FROM scratch

ENV PATH=/opt/cloudflare-access-controller/bin:$PATH

COPY cloudflare-access-controller /opt/cloudflare-access-controller/bin/cloudflare-access-controller

ENTRYPOINT ["/opt/cloudflare-access-controller/bin/cloudflare-access-controller"]