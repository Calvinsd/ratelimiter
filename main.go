package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type TokenBucket struct {
	capacity    int       // capacity of the bucket
	rate        int       // no of tokens to put into bucket every second.
	tokens      int       // no of tokens
	lastUpdated time.Time // keeps track of updated time
	lock        sync.Mutex
}

func (tb *TokenBucket) New(size int, rate int) {
	tb.capacity = size
	tb.rate = rate
	tb.tokens = size
	tb.lastUpdated = time.Now()
}

func (tb *TokenBucket) removeToken() bool {
	tb.lock.Lock()

	defer tb.lock.Unlock()

	tb.refill()

	if tb.tokens > 0 {
		tb.tokens--
		return true
	} else {
		return false
	}
}

func (tb *TokenBucket) refill() {
	elapsedTime := time.Now().Sub(tb.lastUpdated)

	refillTokens := (int(elapsedTime.Seconds()) * tb.rate)

	if refillTokens > 0 {
		tb.lastUpdated = time.Now()
		tb.tokens += refillTokens
	}

	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
}

func main() {
	const port string = ":8080"

	TBucket := TokenBucket{}

	// Initialize token bucket
	TBucket.New(5, 2)

	done := make(chan bool)

	go handleShutdowns(done)

	log.Println("http server on port", port)

	mux := http.NewServeMux()

	mux.HandleFunc("/", homePage)

	// Wrapped ratelimiter middleware
	wrappedMux := rateLimiter(mux, &TBucket)

	// server config
	server := http.Server{
		Addr:         port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      wrappedMux,
	}

	// Start the server
	go func() {
		err := server.ListenAndServe()

		if err != nil {
			log.Fatal(err)
		}

	}()

	log.Println("Wating for shutdowns....")

	<-done

	log.Println("Shutting down")

}

func homePage(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request", r.Host)
	w.Write([]byte("Home Page"))
}

// Rate limiter middleware
func rateLimiter(handler http.Handler, tb *TokenBucket) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardRequest := tb.removeToken()

		if !forwardRequest {
			log.Println("Dropped request from: ", r.Host)
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("429 - Too Many Requests!"))
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// listens for shutdown signals
func handleShutdowns(done chan<- bool) {
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGSEGV)

	go func() {
		sig := <-signalChannel
		switch sig {
		case os.Interrupt:
			log.Println("Encountered os interrupt")
			done <- true
		case syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGSEGV:
			log.Println("Received linux signel")
			done <- true
		}
	}()
}
