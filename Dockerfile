FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN GOOS=linux go build -o /app/cmd/gophermart /build/cmd/gophermart

ENTRYPOINT ["/app/cmd/gophermart"]