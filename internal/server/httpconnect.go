package server

import (
	"bufio"
	"encoding/base64"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/anivaryam/proxy-relay/internal/proxy"
)

func handleHTTPConnect(conn net.Conn, auth *Auth) {
	defer conn.Close()

	br := bufio.NewReader(conn)

	for {
		req, err := http.ReadRequest(br)
		if err != nil {
			return
		}

		if !authenticateHTTP(req, auth) {
			conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"proxy-relay\"\r\n\r\n"))
			return
		}

		if req.Method == http.MethodConnect {
			handleConnect(conn, req)
			return
		}

		handlePlainHTTP(conn, req)
	}
}

func authenticateHTTP(req *http.Request, auth *Auth) bool {
	authHeader := req.Header.Get("Proxy-Authorization")
	if authHeader == "" {
		return false
	}

	// Support Basic auth (browsers)
	if strings.HasPrefix(authHeader, "Basic ") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		if err != nil {
			return false
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return false
		}
		return auth.Validate(parts[1])
	}

	// Support Bearer auth (CLI tools)
	if strings.HasPrefix(authHeader, "Bearer ") {
		return auth.Validate(strings.TrimPrefix(authHeader, "Bearer "))
	}

	return false
}

func handleConnect(conn net.Conn, req *http.Request) {
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

func handlePlainHTTP(conn net.Conn, req *http.Request) {
	host := req.Host
	if !strings.Contains(host, ":") {
		host = host + ":80"
	}

	target, err := net.Dial("tcp", host)
	if err != nil {
		log.Printf("http-proxy: dial %s: %v", host, err)
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer target.Close()

	// Remove proxy headers and forward
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("Proxy-Connection")
	req.RequestURI = req.URL.Path
	if req.URL.RawQuery != "" {
		req.RequestURI += "?" + req.URL.RawQuery
	}

	if err := req.Write(target); err != nil {
		return
	}

	log.Printf("http-proxy: forwarded to %s", host)
	io.Copy(conn, target)
}
