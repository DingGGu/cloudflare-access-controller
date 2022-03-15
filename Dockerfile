FROM alpine as build
RUN apk --no-cache add ca-certificates

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV PATH=/opt/cloudflare-access-controller/bin:$PATH
COPY cloudflare-access-controller /opt/cloudflare-access-controller/bin/cloudflare-access-controller

ENTRYPOINT ["/opt/cloudflare-access-controller/bin/cloudflare-access-controller"]