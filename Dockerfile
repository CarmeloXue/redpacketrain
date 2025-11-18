FROM golang:1.24 as builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/consumer ./cmd/consumer

FROM alpine:3.19
RUN adduser -D -g '' appuser
USER appuser
WORKDIR /home/appuser
COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /out/consumer /usr/local/bin/consumer
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
