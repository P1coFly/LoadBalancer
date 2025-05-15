FROM golang:1.23-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o loudbalancer ./cmd/lb

FROM alpine AS runner

COPY --from=build /app/loudbalancer /loudbalancer
COPY /config/config.yml /config/config.yml

EXPOSE 8080
ENTRYPOINT ["/loudbalancer", "--config=/config.yml"]
