package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	// "github.com/go-playground/statics/static"
	"github.com/GeertJohan/go.rice"
	"github.com/skratchdot/open-golang/open"
)

var pBox *rice.Box

func Handler(response http.ResponseWriter, request *http.Request) {
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

func LaunchServer() int {
	return listenAndServe(0)
}

func LaunchInterface(port int) {
	open.Run(fmt.Sprintf("http://localhost:%d/cover", port))
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
	http.HandleFunc("/", Handler)
	port := LaunchServer()
	fmt.Printf("Port: %d", port)
	LaunchInterface(port)
	for {
		time.Sleep(1 * time.Second)
	}
}
