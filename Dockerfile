FROM golang:latest AS builder
ADD . /app
WORKDIR /app
RUN go mod download

RUN GCO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o build/cloudflare-access-controller

FROM scratch
COPY --from=builder /app/build ./
RUN chmod +x ./cloudflare-access-controller
ENTRYPOINT ["./cloudflare-access-controller"]