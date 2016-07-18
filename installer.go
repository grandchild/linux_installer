package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/cloudfoundry/jibber_jabber"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/text/language"
)

var pBox *rice.Box
var pushChan chan string

func handler(response http.ResponseWriter, request *http.Request) {
	isCommand := handleCommands(response, request.URL.Path, request.URL.Query())
	if isCommand {
		return
	}
	str, err := getResource(request.URL.Path)
	if err != nil && !regexp.MustCompile(`\.[^/]+$`).MatchString(request.URL.Path) {
		str, err = getResource(request.URL.Path + ".html")
	}
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
		fmt.Fprint(response, getAllLanguages())
	default:
		return false
	}
	return true
}

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

func getResource(name string) (string, error) {
	return getResourceFiltered(name, regexp.MustCompile(`.*`))
}
func getResourceFiltered(name string, dirFilter *regexp.Regexp) (string, error) {
	if pBox == nil {
		return "", errors.New(fmt.Sprintf("payload '%s' doesn't exist.", pBox))
	}
	text, err := pBox.String(name)
	if err != nil {
		contents := []string{}
		err = pBox.Walk(name, func(path string, info os.FileInfo, err error) error {
			if path != name {
				if dirFilter.FindStringIndex(path) != nil {
					contents = append(contents, path)
				}
				if info.IsDir() {
					return filepath.SkipDir
				}
			}
			return nil
		})
		if err == nil {
			text = strings.Join(contents, "\n")
		}
	}
	if err != nil {
		return "", errors.New(fmt.Sprint(name, " not found."))
	}
	return text, err
}

func getLocaleMatch() string {
	stringsFiles, _ := getResourceFiltered("strings", regexp.MustCompile(`\.json$`))
	langCodes := strings.Split(regexp.MustCompile(`.*/([^/]+)\.json`).ReplaceAllString(stringsFiles, "$1"), "\n")
	langTags := []language.Tag{language.Raw.Make("en")}
	for _, lang := range langCodes {
		if lang != "en" && lang != "" {
			langTags = append(langTags, language.Raw.Make(lang))
		}
	}
	locale, _ := jibber_jabber.DetectIETF()
	match, _, _ := language.NewMatcher(langTags).Match(language.Make(locale))
	return match.String()
}

func getAllLanguages() string {
	stringsFiles, _ := getResourceFiltered("strings", regexp.MustCompile(`\.json$`))
	langFiles := strings.Split(stringsFiles, "\n")
	langCodes := strings.Split(regexp.MustCompile(`.*/([^/]+)\.json`).ReplaceAllString(stringsFiles, "$1"), "\n")
	langs := []string{}
	for i, lang := range langCodes {
		if lang != "" {
			langStrings, _ := getResource(langFiles[i])
			langs = append(langs, "\""+lang+"\": "+langStrings)
		}
	}
	return "{\n" + strings.Join(langs, ",\n") + "\n}"
}

func main() {
	var err error
	pBox, err = rice.FindBox("payload")
	if err != nil {
		fmt.Println(err)
		return
	}
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
