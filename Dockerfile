FROM alpine
COPY cloudflare-access-controller /
ENTRYPOINT ["./cloudflare-access-controller"]