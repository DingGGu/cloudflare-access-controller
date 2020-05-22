FROM golang:1.14-alpine as builder

RUN apk update && apk add git && apk add make && apk add ca-certificates

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY Makefile Makefile
COPY cmd/ cmd/
COPY internal/ internal/

RUN make build

FROM alpine

ENV PATH=/opt/cloudflare-access-controller/bin:$PATH

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /workspace/cloudflare-access-controller /opt/cloudflare-access-controller/bin/cloudflare-access-controller

ENTRYPOINT ["/opt/cloudflare-access-controller/bin/cloudflare-access-controller"]