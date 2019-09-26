package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/marema31/namecheck/checker"
	"github.com/marema31/namecheck/github"
	"github.com/marema31/namecheck/twitter"
)

// Declare a real http.Client that we will override in tests
var web = http.DefaultClient

// Declare a usage counter that will be updated atomically by the goroutine
var usageCount uint32

// Declare a counter of request for a user that will be updated with mutexes
var nameChecked map[string]uint32 = map[string]uint32{}

// The mutex
var mu sync.Mutex

type availability struct {
	Platform  string `json:socialNetwork`
	Valid     bool
	Available bool
}

type response struct {
	Username     string
	Requested    uint32
	RequestID    uint32
	Availability []availability
}

var checkers []checker.Checker = []checker.Checker{
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
	&twitter.Twitter{},
	&github.Github{},
}

func checkUser(wg *sync.WaitGroup, ch chan availability, username string, c checker.Checker) {
	defer wg.Done()
	var message availability
	message.Platform = c.Name()
	if c.Check(username) {
		message.Valid = true
		available, err := c.IsAvailable(web, username)
		if err != nil {
			log.Printf("No way to contact Twitter: %s", err)

			// On peut aussi redeclarer:
			// type wrapper interface {
			//    Unwrap() error
			//}
			// et remplacer le if par err, ok := err.(wrapper); ok
			if err, ok := err.(interface{ Unwrap() error }); ok {
				log.Fatal(err.Unwrap())
			}
		}
		message.Available = available
	}
	ch <- message
}

func sayHello(w http.ResponseWriter, r *http.Request) {
	calledurl := r.URL.Path
	username := strings.TrimPrefix(calledurl, "/")
	if username == "favicon.ico" {
		return
	}

	mu.Lock()
	//The brace are not mandatory but add a visual indication of the lock section
	{
		nameChecked[username]++
	}
	mu.Unlock()
	count := atomic.AddUint32(&usageCount, 1)
	message := response{Username: username, RequestID: count, Requested: nameChecked[username]}

	var wg sync.WaitGroup

	ch := make(chan availability, 4)
	for _, c := range checkers {
		wg.Add(1)
		go checkUser(&wg, ch, username, c)
	}
	// Must be in goroutine because we can not close the channel before all the checkUser goroutine have finished,
	// but if the chan as a capacity less important than the number of checkUser those will not be able to write to the
	// channel and will block therefore the wg.Wait will never goes backz
	go func() {
		wg.Wait()
		close(ch)
	}()
	for availabilityResponse := range ch {
		message.Availability = append(message.Availability, availabilityResponse)
	}
	messageJSON, _ := json.MarshalIndent(message, "", "    ")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(messageJSON))
}

func main() {
	http.HandleFunc("/", sayHello)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
