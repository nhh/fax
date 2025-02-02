package main

// https://github.com/kunif/EscPosUtils

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nhh/fax/internal/limiter"
	"github.com/nhh/fax/internal/orm"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"
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

var rl = limiter.New()

type Message struct {
	author string
	text   string
	time   time.Time
}

//go:embed schema.sql
var ddl string

var queries *orm.Queries
var env *string

func init() {
	fmt.Println(ddl)
	ctx := context.TODO()

	db, err := sql.Open("sqlite3", "./fax.db")
	if err != nil {
		panic(err)
	}

	// create tables
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		panic(err)
	}

	queries = orm.New(db)
}

func main() {
	env = flag.String("env", "production", "Set stuff to dev")
	listenAddress := flag.String("listen", ":3000", "Port to listen for incoming connections.")
	flag.Parse()

	mux := http.NewServeMux()

	fmt.Printf("Running in %s...", *env)

	if *env == "development" {
		mux.Handle("GET /static/", http.FileServer(http.Dir("static")))

		mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "./index.html")
		})
	} else {
		mux.Handle("GET /static/", http.FileServer(http.FS(embedDirStatic)))

		mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			// Every request gets a cookie, that is http only to avoid cross origin request forgery
			cookie := http.Cookie{
				Name:     "plsdonthackme",
				Value:    "pls",
				Domain:   "fax.pfusch.dev",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}

			http.SetCookie(w, &cookie)

			http.ServeFileFS(w, r, f, "index.html")
		})
	}

	mux.HandleFunc("POST /fax", handleFax)

	log.Fatal(http.ListenAndServe(*listenAddress, mux))
}

func handleFax(w http.ResponseWriter, r *http.Request) {
	if rl.IsLimited() {
		w.WriteHeader(429)
		return
	}

	// Using a http only cookie to verify, that request came from the correct origin
	_, err := r.Cookie("plsdonthackme")

	if err != nil && *env == "production" {
		fmt.Println(err)
		http.Redirect(w, r, "/", 302)
		return
	}

	rl.Increment()

	text := r.FormValue("text")
	name := r.FormValue("name")

	if text == "" || utf8.RuneCountInString(text) < 20 {
		w.WriteHeader(400)
		return
	}

	if utf8.RuneCountInString(text) > 512 {
		http.Redirect(w, r, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", 303)
		return
	}

	if utf8.RuneCountInString(name) > 128 {
		http.Redirect(w, r, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", 303)
		return
	}

	currentTime := time.Now().UTC()

	fmt.Println("~~~~~~ MESSAGE ~~~~~~")
	fmt.Println(currentTime.String())
	fmt.Println(text)
	fmt.Println("")
	fmt.Println(name)
	fmt.Println("~~~~~~ END ~~~~~~")

	// fire and forget
	go sendToPrinter(Message{name, text, currentTime})
	go saveToDb(orm.CreateAuthorParams{
		Name: name,
		Bio:  sql.NullString{String: text, Valid: true},
	})

	count, err := queries.CountAuthors(context.TODO())
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(count)

	http.Redirect(w, r, "/", 302)
}

func saveToDb(params orm.CreateAuthorParams) {
	_, err := queries.CreateAuthor(context.TODO(), params)
	if err != nil {
		fmt.Println(err)
	}
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
