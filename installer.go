package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/skratchdot/open-golang/open"
)

var pushChan chan string

func LaunchServer() int {
	return listenAndServe(0)
}

func LaunchInterface(port int) {
	open.Run(fmt.Sprintf("http://localhost:%d/install", port))
}

// The SignalHandler sets up a channel to wait for an interrupt sent to the
// program (Ctrl+C on the command line, most likely).
// It then fires a goroutine which sends a "quit" command to the pushChan
// channel, which in turn will send it to the interface, which will close or
// deactivate the browser tab.
func LaunchSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		pushChan <- "quit"
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
}

// The Push request is a longstanding, continually refreshing request from the
// interface which lets this server send commands back to the interface (while
// normally the server can only answer to requests). Any string sent to the
// pushChan go channel object will be sent to the interface as is and needs to
// be handled there.
func LaunchPush(response http.ResponseWriter) {
	select {
	case pushCommand := <-pushChan:
		fmt.Fprint(response, pushCommand)
	case <-time.After(1 * time.Minute):
		fmt.Fprint(response, "refresh push")
	}
}

func main() {
	openBoxes()
	pushChan = make(chan string, 1)
	http.HandleFunc("/", handler)
	port := LaunchServer()
	LaunchInterface(port)
	LaunchSignalHandler()
	fmt.Printf("Port: %d\n", port)
	for {
		time.Sleep(1 * time.Second)
	}
}
