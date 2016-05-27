package main

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func listenAndServe(port int) int {
	if port < 0 {
		return 0
	}
	srv := http.Server{
		Addr: fmt.Sprintf("localhost:%d", port),
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return 0
	}
	_, gotPort, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return 0
	}
	gotPortI, err := strconv.Atoi(gotPort)
	if err != nil {
		return 0
	}
	go srv.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
	return gotPortI
}
