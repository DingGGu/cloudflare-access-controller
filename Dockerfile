FROM alpine
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY cloudflare-access-controller /
ENTRYPOINT ["./cloudflare-access-controller"]