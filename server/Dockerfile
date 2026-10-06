FROM golang:1.25.6-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/service ./cmd/api

FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/service /bin/service
EXPOSE 3000
ENTRYPOINT ["/bin/service"]
