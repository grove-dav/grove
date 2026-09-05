# syntax=docker/dockerfile:1
# SPDX-FileCopyrightText: 2026 Grove contributors
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.27.1-bookworm@sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b AS build
WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN go build -trimpath -ldflags="-s -w" -o /out/grove ./cmd/grove

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
COPY --from=build /out/grove /grove
EXPOSE 8080
ENTRYPOINT ["/grove"]
