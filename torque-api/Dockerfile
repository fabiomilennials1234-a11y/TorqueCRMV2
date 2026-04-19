# syntax=docker/dockerfile:1.7
# -----------------------------------------------------------------------------
# Torque API — multi-stage build.
#   Stage 1: builder  — compiles a static binary with the build toolchain.
#   Stage 2: runtime  — distroless, no shell, minimal attack surface.
# -----------------------------------------------------------------------------

ARG GO_VERSION=1.22

# -------- builder ------------------------------------------------------------
FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ARG VERSION=dev
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/torque-api \
      ./cmd/api

# -------- runtime ------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo              /usr/share/zoneinfo
COPY --from=builder /out/torque-api                  /torque-api

EXPOSE 8080

# nonroot user (uid 65532) is baked in by distroless.
USER nonroot:nonroot

ENTRYPOINT ["/torque-api"]
