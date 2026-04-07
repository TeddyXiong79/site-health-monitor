# syntax=docker/dockerfile:1

# ===== Stage 1: Build =====
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

WORKDIR /src

# Cache go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Cross-compile: TARGETOS/TARGETARCH passed by buildx, defaults to linux/amd64
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -trimpath \
    -o site-health-monitor .

# ===== Stage 2: Distroless runtime =====
FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:nonroot

# Copy static binary (position at root so base image CMD can find it)
COPY --from=builder /src/site-health-monitor /site-health-monitor

# Copy default config.json (bind-mount a new one at runtime to override)
COPY config.json /config.json

USER root
RUN chmod 0444 /config.json
USER nonroot

EXPOSE 9099

ENTRYPOINT ["/site-health-monitor"]
