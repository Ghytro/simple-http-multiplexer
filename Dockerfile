FROM golang:1.18.3-alpine as builder
WORKDIR /app
COPY . .
RUN cd cmd/simple-http-handler && go build -o app

FROM alpine
COPY --from=builder /app/cmd/simple-http-handler/app ./app
EXPOSE 8080
ENTRYPOINT ["/app"]
