# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

FROM golang:1.18 AS builder
ARG VERSION

ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /src
COPY . .

RUN go build \
  -ldflags "-X 'github.com/hashicorp/tfci/version.Version=$VERSION' -s -w -extldflags '-static'" \
  -o /bin/app \
  .

FROM alpine:latest

COPY --from=builder /bin/app /usr/local/bin/tfci

ENTRYPOINT []
