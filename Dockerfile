# Build and run the backend (for Dokku; build context = repo root)
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=0 go build -o /backend .

FROM alpine:3.19

RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /backend .

EXPOSE 8080

CMD ["./backend"]
