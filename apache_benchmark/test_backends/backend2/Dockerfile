FROM golang:1.23-alpine AS build

WORKDIR /app
COPY . .
RUN go build -o backend2 ./cmd

FROM alpine AS runner
COPY --from=build /app/backend2 /backend2

EXPOSE 8082
ENTRYPOINT ["/backend2"]
