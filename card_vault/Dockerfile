FROM golang:1.27-alpine AS builder
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /card-vault ./cmd/server

FROM alpine:3.24
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /card-vault .
EXPOSE 8080
ENTRYPOINT ["./card-vault"]