FROM golang:1.23-alpine AS build

WORKDIR /app
COPY . .
RUN go build -o backend1 ./cmd

FROM alpine AS runner
COPY --from=build /app/backend1 /backend1

EXPOSE 8081
ENTRYPOINT ["/backend1"]
