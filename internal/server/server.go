package server

import (
	"bufio"
	"log"
	"net"
)

type bufferedConn struct {
	r *bufio.Reader
	net.Conn
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

type Server struct {
	addr     string
	auth     *Auth
	listener net.Listener
}

func New(addr string, auth *Auth) *Server {
	return &Server{addr: addr, auth: auth}
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = ln
	log.Printf("proxy server listening on %s", s.addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ne, ok := err.(*net.OpError); ok && !ne.Temporary() {
				return nil
			}
			log.Printf("accept error: %v", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) handleConn(conn net.Conn) {
	br := bufio.NewReader(conn)
	bc := &bufferedConn{r: br, Conn: conn}

	first, err := br.Peek(1)
	if err != nil {
		conn.Close()
		return
	}

	if first[0] == 0x05 {
		handleSOCKS5(bc, s.auth)
	} else {
		handleHTTPConnect(bc, s.auth)
	}
}
