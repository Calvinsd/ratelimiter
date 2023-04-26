package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

func main() {

	var wg sync.WaitGroup

	// Number of requests
	const size int = 20

	wg.Add(size)

	for i := 0; i < size; i++ {
		go sendRequest("http://localhost:8080", &wg)

		// Delay
		time.Sleep(200 * time.Millisecond)
	}

	wg.Wait()

}

func sendRequest(endpoint string, wg *sync.WaitGroup) {
	resp, err := http.Get(endpoint)

	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		log.Println("Request droped by ratelimiter", resp.StatusCode)
	} else {
		log.Println("Request successfull", resp.StatusCode)
	}

	wg.Done()
}
