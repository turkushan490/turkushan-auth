// Package mailtest is a tiny fake SMTP server that keeps what it receives, for tests.
package mailtest

import (
	"bufio"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"strings"
	"sync"
	"testing"
)

type Server struct {
	Host string
	Port string
	ln   net.Listener
	mu   sync.Mutex
	msgs []string
}

// Start listens on a random loopback port until the test ends.
func Start(t testing.TB) *Server {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	s := &Server{Host: host, Port: port, ln: ln}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handle(c)
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return s
}

func (s *Server) handle(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	reply := func(line string) { io.WriteString(c, line+"\r\n") }
	reply("220 fake ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			reply("250-fake")
			reply("250 AUTH PLAIN")
		case strings.HasPrefix(cmd, "AUTH"):
			reply("235 ok")
		case cmd == "DATA":
			reply("354 go ahead")
			var body strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				body.WriteString(strings.TrimPrefix(l, "."))
			}
			s.mu.Lock()
			s.msgs = append(s.msgs, body.String())
			s.mu.Unlock()
			reply("250 queued")
		case cmd == "QUIT":
			reply("221 bye")
			return
		default:
			reply("250 ok")
		}
	}
}

// Count returns how many messages arrived.
func (s *Server) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.msgs)
}

// Last returns the To header and decoded text part of the newest message.
func (s *Server) Last(t testing.TB) (to, text string) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.msgs) == 0 {
		t.Fatal("no mail received")
	}
	msg, err := mail.ReadMessage(strings.NewReader(s.msgs[len(s.msgs)-1]))
	if err != nil {
		t.Fatal(err)
	}
	_, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	part, err := multipart.NewReader(msg.Body, params["boundary"]).NextPart() // decodes quoted-printable
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(part)
	return msg.Header.Get("To"), string(b)
}
