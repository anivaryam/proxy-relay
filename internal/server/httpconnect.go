package server

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/anivaryam/proxy-relay/internal/proxy"
)

func handleHTTPConnect(conn net.Conn, auth *Auth) {
	defer conn.Close()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		return
	}

	// Authenticate
	authHeader := req.Header.Get("Proxy-Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if authHeader == "" || !auth.Validate(token) {
		conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Bearer\r\n\r\n"))
		return
	}

	if req.Method != http.MethodConnect {
		conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
		return
	}

	target, err := net.Dial("tcp", req.Host)
	if err != nil {
		log.Printf("http-connect: dial %s: %v", req.Host, err)
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	log.Printf("http-connect: connected to %s", req.Host)
	proxy.Relay(conn, target)
}
