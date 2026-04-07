package server

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/anivaryam/proxy-relay/internal/proxy"
)

const (
	socks5Version  = 0x05
	authUserPass   = 0x02
	authNoAccept   = 0xFF
	userPassVer    = 0x01
	cmdConnect     = 0x01
	atypIPv4       = 0x01
	atypDomain     = 0x03
	atypIPv6       = 0x04
	repSuccess     = 0x00
	repFailure     = 0x01
	repCmdNotSupp  = 0x07
	repAddrNotSupp = 0x08
)

func handleSOCKS5(conn net.Conn, auth *Auth) {
	defer conn.Close()

	// 1. Greeting
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return
	}
	if header[0] != socks5Version {
		return
	}
	methods := make([]byte, header[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}

	// Require username/password auth
	hasUserPass := false
	for _, m := range methods {
		if m == authUserPass {
			hasUserPass = true
			break
		}
	}
	if !hasUserPass {
		conn.Write([]byte{socks5Version, authNoAccept})
		return
	}
	conn.Write([]byte{socks5Version, authUserPass})

	// 2. Username/password auth (RFC 1929)
	if _, err := io.ReadFull(conn, header[:1]); err != nil {
		return
	}
	if header[0] != userPassVer {
		return
	}

	// Read username (ignored, token is in password)
	if _, err := io.ReadFull(conn, header[:1]); err != nil {
		return
	}
	uname := make([]byte, header[0])
	if _, err := io.ReadFull(conn, uname); err != nil {
		return
	}

	// Read password (this is the auth token)
	if _, err := io.ReadFull(conn, header[:1]); err != nil {
		return
	}
	passwd := make([]byte, header[0])
	if _, err := io.ReadFull(conn, passwd); err != nil {
		return
	}

	if !auth.Validate(string(passwd)) {
		conn.Write([]byte{userPassVer, 0x01}) // auth failure
		return
	}
	conn.Write([]byte{userPassVer, 0x00}) // auth success

	// 3. Connect request
	req := make([]byte, 4)
	if _, err := io.ReadFull(conn, req); err != nil {
		return
	}
	if req[0] != socks5Version {
		return
	}
	if req[1] != cmdConnect {
		sendSOCKS5Reply(conn, repCmdNotSupp)
		return
	}

	addr, err := readSOCKS5Addr(conn, req[3])
	if err != nil {
		sendSOCKS5Reply(conn, repAddrNotSupp)
		return
	}

	// Dial target
	target, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("socks5: dial %s: %v", addr, err)
		sendSOCKS5Reply(conn, repFailure)
		return
	}

	sendSOCKS5Reply(conn, repSuccess)
	log.Printf("socks5: connected to %s", addr)
	proxy.Relay(conn, target)
}

func readSOCKS5Addr(r io.Reader, atyp byte) (string, error) {
	var host string
	switch atyp {
	case atypIPv4:
		buf := make([]byte, 4)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		host = net.IP(buf).String()
	case atypDomain:
		buf := make([]byte, 1)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		domain := make([]byte, buf[0])
		if _, err := io.ReadFull(r, domain); err != nil {
			return "", err
		}
		host = string(domain)
	case atypIPv6:
		buf := make([]byte, 16)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		host = net.IP(buf).String()
	default:
		return "", fmt.Errorf("unsupported address type: %d", atyp)
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, portBuf); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portBuf)
	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}

func sendSOCKS5Reply(conn net.Conn, rep byte) {
	// VER, REP, RSV, ATYP(IPv4), BIND.ADDR(0.0.0.0), BIND.PORT(0)
	conn.Write([]byte{socks5Version, rep, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
}
