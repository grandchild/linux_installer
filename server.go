package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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

func handler(response http.ResponseWriter, request *http.Request) {
	isCommand := handleCommands(response, request.URL.Path, request.URL.Query())
	if isCommand {
		return
	}
	str, err := getInstallerResource(request.URL.Path)
	if err != nil && !regexp.MustCompile(`\.[^/]+$`).MatchString(request.URL.Path) {
		str, err = getInstallerResource(request.URL.Path + ".html")
	}
	if err != nil {
		fmt.Println(err)
		response.WriteHeader(http.StatusNotFound)
		return
	}
	if strings.HasSuffix(request.URL.Path, ".css") {
		response.Header().Set("Content-Type", "text/css;\ncharset=UTF-8")
	}
	if regexp.MustCompile(`.*\.(json|conf(ig)?)$`).MatchString(request.URL.Path) {
		response.Header().Set("Content-Type", "application/json;\ncharset=UTF-8")
	}
	fmt.Fprint(response, str)
}

func handleCommands(response http.ResponseWriter, path string, params url.Values) bool {
	switch path {
	case "/quit":
		os.Exit(0)
	case "/os":
		fmt.Fprint(response, runtime.GOOS)
	case "/locale":
		fmt.Fprint(response, getLocaleMatch())
	case "/copy":
		fmt.Println("src: " + params.Get("src") + " ---> dst: " + params.Get("dst"))
	case "/push":
		// fmt.Println("Push channel opened.")
		LaunchPush(response)
	case "/strings":
		response.Header().Set("Content-Type", "application/json;\ncharset=UTF-8")
		fmt.Fprint(response, getAllLanguages())
	default:
		return false
	}
	return true
}
