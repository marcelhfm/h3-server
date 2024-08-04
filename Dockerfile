FROM golang:1.22.2-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . . 

RUN CGO_ENABLED=1 GOOS=linux go build -o h3-server .

FROM alpine:latest

WORKDIR /root/

RUN apk add --no-cache libc6-compat

COPY --from=builder /app/h3-server .

EXPOSE 5005

CMD ["./h3-server"]
