FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGET=control
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/app ./cmd/${TARGET}

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache bash ca-certificates python3 py3-pip nodejs npm
RUN adduser -D -u 10001 appuser

COPY --from=builder /out/app /app/app

EXPOSE 8180 4280 9190
ENTRYPOINT ["/app/app"]
