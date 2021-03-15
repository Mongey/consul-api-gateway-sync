FROM --platform=$BUILDPLATFORM golang:1.16.1 AS builder

WORKDIR /go/src/github.com/Mongey/consul-api-gateway-sync

COPY . ./
ENV CGO_ENABLED=0
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build .

FROM --platform=$BUILDPLATFORM alpine
COPY  --from=builder /go/src/github.com/Mongey/consul-api-gateway-sync/consul-api-gateway-sync /usr/local/bin
ENTRYPOINT ["/usr/local/bin/consul-api-gateway-sync"]
