package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"net/url"
)

// maxRetries indicates the maximum amount of retries we will perform before
// giving up
var maxRetries = 1

// mirrorRequest will POST through body and headers from an
// incoming http.Request.
// Failures are retried up to 10 times.
func mirrorRequest(h http.Header, body []byte, u string) {
	attempt := 1
	for {
		fmt.Printf("Attempting %s try=%d\n", u, attempt)

		client := &http.Client{}

		rB := bytes.NewReader(body)
		req, err := http.NewRequest("POST", u, rB)
		if err != nil {
			log.Println("[error] http.NewRequest:", err)
		}

		// Set headers from request
		req.Header = h

		resp, err := client.Do(req)
		if err != nil {
			log.Println("[error] client.Do:", err)
			time.Sleep(10 * time.Second)
		} else {
			resp.Body.Close()
			fmt.Printf("[success] %s status=%d\n", u, resp.StatusCode)
			break
		}

		attempt++
		if attempt > maxRetries {
			fmt.Println("[error] maxRetries reached")
			break
		}
	}
}

// parseSites gets sites out of the FORWARDHOOK_SITES environment variable.
// There is no validation at the moment but you can add 1 or more sites,
// separated by commas.
func parseSites() []string {

	contents, _ := ioutil.ReadFile("sites")
	sites := strings.TrimSpace(string(contents))

	s := strings.Split(sites, "\n")

	return s
}

func handleHook(sites []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		rB, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Fail on ReadAll")
		}
		r.Body.Close()

		q := r.URL.Query()

		for _, site := range sites {
			u, _ := url.Parse(site)
			u.RawQuery = q.Encode()
			go mirrorRequest(r.Header, rB, u.String())
		}

		w.WriteHeader(http.StatusOK)
	})
}

func determineListenAddress() (string, error) {
 	port := os.Getenv("PORT")
	if port == "" {
    	return ":8001", nil
  	}
  	return ":" + port, nil
}

func main() {
	sites := parseSites()
	fmt.Println("Will forward hooks to:", sites)

	http.Handle("/", handleHook(sites))

	port, _ := determineListenAddress()

	fmt.Printf("Listening on port %s \n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
