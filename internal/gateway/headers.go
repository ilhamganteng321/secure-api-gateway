package gateway

import (
	"net/http"
	"strings"
)

var blockedRequestHeaders = map[string]bool{
	"Forwarded":          true,
	"X-Real-IP":          true,
	"X-Forwarded-For":    true,
	"X-Forwarded-Host":   true,
	"X-Forwarded-Proto":  true,
	"X-Gateway-Upstream": true,
	"X-Original-URL":     true,
	"X-Rewrite-URL":      true,
}

func SanitizeRequestHeaders(header http.Header) {
	for _, value := range header.Values("Connection") {
		for _, token := range strings.Split(value, ",") {
			header.Del(strings.TrimSpace(token))
		}
	}

	header.Del("Connection")

	for name := range blockedRequestHeaders {
		header.Del(name)
	}

	header.Set("X-Gateway", "secure-api-gateway")
}
