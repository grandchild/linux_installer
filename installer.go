package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	// "github.com/go-playground/statics/static"
	"github.com/GeertJohan/go.rice"
	"github.com/skratchdot/open-golang/open"
)

var pBox *rice.Box

func Handler(response http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(response, request.URL.Path)
	fmt.Fprintf(response, "\n")
	str, err := getResource(pBox, request.URL.Path)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Fprintf(response, str)
}

func LaunchServer() int {
	return listenAndServe(0)
}

func LaunchInterface(port int) {
	open.Run(fmt.Sprintf("http://localhost:%d/hey.txt", port))
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
	pBox, err := rice.FindBox("payload")

	str, err := getResource(pBox, "text.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(str)
	http.HandleFunc("/", Handler)
	port := LaunchServer()
	LaunchInterface(port)
	for {
		time.Sleep(1 * time.Second)
	}
}
