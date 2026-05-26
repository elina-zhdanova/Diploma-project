# Build
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/itshop-api ./cmd/api

# Run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /bin/itshop-api .
COPY migrations ./migrations
EXPOSE 8080
ENV HTTP_PORT=8080
CMD ["./itshop-api"]
