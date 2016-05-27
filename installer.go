package main

import (
	"fmt"
	"net/http"

	// "github.com/go-playground/statics/static"
	"github.com/GeertJohan/go.rice"
	"github.com/skratchdot/open-golang/open"
)

func Handler(response http.ResponseWriter, request *http.Request) {
	fmt.Printf("got a request")
	fmt.Fprintf(response, "Hey!")
}

func LaunchServer() int {
	return listenAndServe(0)
}

func LaunchInterface(port int) {
	open.Run(fmt.Sprintf("http://localhost:%d/", port))
}

func getPayload() string {
	fileBox, err := rice.FindBox("payload")
	if err != nil {
		fmt.Print(err)
		return "Error"
	}
	text, err := fileBox.String("text.txt")
	if err != nil {
		fmt.Print(err)
		return "Error"
	}
	return text
}

func main() {
	fmt.Print(getPayload())
	http.HandleFunc("/", Handler)
	port := LaunchServer()
	LaunchInterface(port)
	for {
	}
}
