package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

var (
	Port,
	MaxIncomingConnections,
	MaxUrlPerRequest,
	MaxOutcomingRequestsPerRequest int

	UrlRequestTimeout,
	RequestHandleTimeout time.Duration
)

func parseInt(val string) int {
	result, err := strconv.Atoi(val)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

func getConfigKey(key string) string {
	return os.Getenv("SIMPLE_HTTP_MUX_" + key)
}

func getIntConfigKey(key string) int {
	return parseInt(getConfigKey(key))
}

func init() {
	Port = getIntConfigKey("PORT")
	MaxIncomingConnections = getIntConfigKey("MAX_INCOMING_CONNS")
	MaxUrlPerRequest = getIntConfigKey("MAX_URL_PER_REQUEST")
	MaxOutcomingRequestsPerRequest = getIntConfigKey("MAX_OUTCOMING_REQUESTS_PER_REQUEST")
	UrlRequestTimeout = time.Duration(getIntConfigKey("URL_REQUEST_TIMEOUT"))
	RequestHandleTimeout = time.Duration(getIntConfigKey("REQUEST_HANDLE_TIMEOUT"))
}
