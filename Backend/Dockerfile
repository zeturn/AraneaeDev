# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.25-alpine AS builder
WORKDIR /src

# Download modules first so this layer is cached unless go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Build the selected target binary (control | executor).
COPY . .
ARG TARGET=control
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -o /out/app ./cmd/${TARGET}

# ---- Runtime stage ----
FROM alpine:3.20 AS runtime
WORKDIR /app

# Base runtime deps shared by both targets.
RUN apk add --no-cache ca-certificates bash

# Only the executor may run user-provided tasks that need python/node;
# the control plane does not need them, so keep its image lean.
ARG TARGET=control
RUN if [ "$TARGET" = "executor" ]; then \
        apk add --no-cache python3 py3-pip nodejs npm; \
    fi

COPY --from=builder /out/app /app/app

# 8180 = control HTTP, 9190 = control gRPC, 4280 = executor HTTP
EXPOSE 8180 4280 9190
ENTRYPOINT ["/app/app"]
