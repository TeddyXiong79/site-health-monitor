# Stage 1: Build
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s" \
    -trimpath \
    -o site-health-monitor .

# Stage 2: scratch (empty, ultra-minimal)
FROM scratch

COPY --from=builder /src/site-health-monitor /site-health-monitor
COPY --from=builder /src/config.json /config.json

EXPOSE 9099

ENTRYPOINT ["/site-health-monitor"]
