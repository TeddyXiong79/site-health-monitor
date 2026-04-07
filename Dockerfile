# syntax=docker/dockerfile:1.4
# enable BuildKit multi-platform features

# ===== Stage 1: Build =====
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

WORKDIR /src

# Copy go.mod and download deps first (layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -trimpath \
    -o site-health-monitor .

# ===== Stage 2: Final runtime =====
FROM --platform=$TARGETPLATFORM gcr.io/distroless/static:nonroot

# Binary at root so base image ENTRYPOINT finds it
COPY --from=builder /src/site-health-monitor /

# Default config — bind-mount a custom config.json at runtime to override
COPY config.json /config.json

EXPOSE 9099

ENTRYPOINT ["/site-health-monitor"]
