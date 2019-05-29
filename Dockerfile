FROM scratch
COPY cloudflare-access-controller /
ENTRYPOINT ["./cloudflare-access-controller"]