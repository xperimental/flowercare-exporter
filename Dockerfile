FROM --platform=$BUILDPLATFORM golang:1.19.5-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

RUN apk add --no-cache make git bash

WORKDIR /build

COPY go.mod go.sum /build/
RUN go mod download
RUN go mod verify

COPY . /build/
RUN make build-binary

FROM --platform=$TARGETPLATFORM busybox
LABEL maintainer="Robert Jacob <xperimental@solidproject.de>"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /build/flowercare-exporter /bin/flowercare-exporter

USER nobody
EXPOSE 9294

ENTRYPOINT ["/bin/flowercare-exporter"]
