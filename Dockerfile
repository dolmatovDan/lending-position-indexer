FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/positions ./cmd/positions

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
USER app
COPY --from=build /out/positions /usr/local/bin/positions
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/positions"]
