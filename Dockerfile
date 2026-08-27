# syntax=docker/dockerfile:1
# SPDX-FileCopyrightText: 2026 Grove contributors
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36 AS build
WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN go build -trimpath -ldflags="-s -w" -o /out/grove ./cmd/grove

FROM gcr.io/distroless/static-debian12:nonroot@sha256:1b7b9f0f0e0a1d2155f531db587cc48ec26aaf97ab64364225f5bf18a054e66a
COPY --from=build /out/grove /grove
EXPOSE 8080
ENTRYPOINT ["/grove"]
