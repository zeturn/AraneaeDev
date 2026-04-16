FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGET=control
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app ./cmd/${TARGET}

FROM alpine:3.20
WORKDIR /app
RUN adduser -D -u 10001 appuser

COPY --from=builder /out/app /app/app
USER appuser

EXPOSE 8180 4280 9190
ENTRYPOINT ["/app/app"]
