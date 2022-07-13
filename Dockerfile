FROM golang:1.18.3-alpine as build
WORKDIR /app
COPY . .
RUN cd cmd/simple-http-handler && go build -o app

FROM build as test
RUN apk add build-base
ENV SIMPLE_HTTP_MUX_MAX_INCOMING_CONNS=100 \
    SIMPLE_HTTP_MUX_MAX_URL_PER_REQUEST=20 \
    SIMPLE_HTTP_MUX_MAX_OUTCOMING_REQUESTS_PER_REQUEST=4 \
    SIMPLE_HTTP_MUX_URL_REQUEST_TIMEOUT=5 \
    SIMPLE_HTTP_MUX_REQUEST_HANDLE_TIMEOUT=20
RUN go test ./test/

FROM alpine as prod
COPY --from=build /app/cmd/simple-http-handler/app ./app
EXPOSE 8080
ENTRYPOINT ["/app"]
