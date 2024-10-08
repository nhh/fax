package main

// https://github.com/kunif/EscPosUtils

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// Embed a single file
//
//go:embed index.html
var f embed.FS

// Embed a directory
//
//go:embed static/*
var embedDirStatic embed.FS

var lock sync.Mutex

type RateLimit struct {
	mu      sync.Mutex
	counter int
}

var limiter = RateLimit{
	counter: 0,
}

func (limiter *RateLimit) IsLimited() bool {
	return limiter.counter >= 5
}

type Message struct {
	author string
	text   string
	time   time.Time
}

func main() {
	env := flag.String("env", "production", "Set stuff to dev")
	listenAddress := flag.String("listen", ":3000", "Port to listen for incoming connections.")
	flag.Parse()

	mux := http.NewServeMux()

	// Shoot off ratelimiter to infinity
	go resetRateLimit()

	fmt.Printf("Running in %s...", *env)

	if *env == "development" {
		mux.Handle("GET /static/", http.FileServer(http.Dir("static")))

		mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./index.html")
		})
	} else {
		mux.Handle("GET /static/", http.FileServer(http.FS(embedDirStatic)))

		mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, f, "index.html")
		})
	}

	mux.HandleFunc("POST /fax", handleFax)

	log.Fatal(http.ListenAndServe(*listenAddress, mux))
}

func resetRateLimit() {
	for {
		time.Sleep(60 * 5 * time.Second)
		limiter.mu.Lock()
		limiter.counter = 0
		limiter.mu.Unlock()
	}
}

func handleFax(w http.ResponseWriter, r *http.Request) {
	if limiter.IsLimited() {
		w.WriteHeader(429)
		return
	}

	limiter.mu.Lock()
	limiter.counter++
	limiter.mu.Unlock()

	text := r.FormValue("text")
	name := r.FormValue("name")
	currentTime := time.Now().UTC()

	fmt.Println("~~~~~~ MESSAGE ~~~~~~")
	fmt.Println(currentTime.String())
	fmt.Println(text)
	fmt.Println("")
	fmt.Println(name)
	fmt.Println("~~~~~~ END ~~~~~~")

	// fire and forget
	go sendToPrinter(Message{name, text, currentTime})

	http.Redirect(w, r, "/", 302)
}

func sendToPrinter(msg Message) {
	lock.Lock()
	defer lock.Unlock()

	conn, err := net.Dial("tcp", "192.168.188.232:9100")
	if err != nil {
		return
	}
	defer closeWithLog(conn)

	conn.Write([]byte{0x1B, 0x40}) // Initialize
	conn.Write([]byte("~~~~~~ MESSAGE ~~~~~~\n"))
	conn.Write([]byte(msg.time.String()))
	conn.Write([]byte("\n"))
	conn.Write([]byte("\n"))
	conn.Write([]byte(msg.text))
	conn.Write([]byte("\n"))
	conn.Write([]byte(msg.author))
	conn.Write([]byte("\n"))
	conn.Write([]byte("~~~~~~ END ~~~~~~\n"))
}

func closeWithLog(closable Closable) {
	err := closable.Close()
	if err != nil {
		log.Println("error: " + err.Error())
	}
}

type Closable interface {
	Close() error
}
