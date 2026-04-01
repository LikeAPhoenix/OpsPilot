FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/server /app/server
COPY configs /app/configs

ENV OPSPILOT_CONFIG=/app/configs/config.yaml

EXPOSE 6872

CMD ["/app/server"]
