package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	// "github.com/go-playground/statics/static"
	"github.com/GeertJohan/go.rice"
	"github.com/skratchdot/open-golang/open"
)

var pBox *rice.Box

func handler(response http.ResponseWriter, request *http.Request) {
	isCommand, err := handleCommands(request.URL.Path, request.URL.Query())
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(response, err)
	}
	if isCommand {
		emptyOK(response)
		return
	}
	str, err := getResource(pBox, request.URL.Path)
	if err != nil {
		fmt.Println(err)
		response.WriteHeader(http.StatusNotFound)
		return
	}
	if strings.HasSuffix(request.URL.Path, ".css") {
		response.Header().Set("Content-Type", "text/css")
	}
	fmt.Fprint(response, str)
}

func emptyOK(response http.ResponseWriter) {
	fmt.Fprint(response)
}

func handleCommands(path string, params url.Values) (bool, error) {
	var err error
	switch path {
	case "/quit":
		os.Exit(0)
		return true, err
	case "/copy":
		fmt.Println("src: " + params.Get("src") + " ---> dst: " + params.Get("dst"))
		return true, err
	default:
		return false, err
	}

}

func LaunchServer() int {
	return listenAndServe(0)
}

func LaunchInterface(port int) {
	open.Run(fmt.Sprintf("http://localhost:%d/install", port))
}

func getResource(box *rice.Box, name string) (string, error) {
	if box == nil {
		return "", errors.New(fmt.Sprintf("payload '%s' doesn't exist.", box))
	}
	text, err := box.String(name)
	if err != nil {
		return "", errors.New(fmt.Sprintf("resource '%s' not found.", name))
	}
	return text, err
}

func main() {
	var err error
	pBox, err = rice.FindBox("payload")
	if err != nil {
		fmt.Println(err)
		return
	}
	http.HandleFunc("/", handler)
	port := LaunchServer()
	fmt.Printf("Port: %d\n", port)
	LaunchInterface(port)
	for {
		time.Sleep(1 * time.Second)
	}
}
