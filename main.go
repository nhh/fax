package main

// https://github.com/kunif/EscPosUtils

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// Embed a single file
//
//go:embed index.html
var f embed.FS

// Embed a directory
//
//go:embed static/*
var embedDirStatic embed.FS

func main() {
	env := flag.String("env", "production", "Set stuff to dev")
	flag.Parse()

	app := fiber.New()

	// Or extend your config for customization
	app.Use(limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			// Only limit actual fax submits
			return c.Path() != "/fax"
		},
		Max:        5,
		Expiration: 5 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.Get("X-Original-Forwarded-For") // Stands behind cloudflare
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.SendStatus(429)
		},
	}))

	if *env == "development" {
		app.Get("/", func(ctx *fiber.Ctx) error {
			return ctx.SendFile("./index.html")
		})

		app.Static("/static", "./static")
	} else {
		app.Use("/", filesystem.New(filesystem.Config{
			Root: http.FS(f),
		}))

		// Access file "image.png" under `static/` directory via URL: `http://<server>/static/image.png`.
		// Without `PathPrefix`, you have to access it via URL:
		// `http://<server>/static/static/image.png`.
		app.Use("/static", filesystem.New(filesystem.Config{
			Root:       http.FS(embedDirStatic),
			PathPrefix: "static",
			Browse:     true,
		}))
	}

	app.Post("/fax", handleFax)
	log.Fatal(app.Listen(":3000"))
}

func handleFax(ctx *fiber.Ctx) error {
	text := ctx.FormValue("text")
	name := ctx.FormValue("name")

	go func() {
		conn, err := net.Dial("tcp", "192.168.188.232:9100")
		if err != nil {
			return
		}
		defer conn.Close()

		fmt.Println("~~~~~~ MESSAGE ~~~~~~")
		fmt.Println(time.Now().UTC().String())
		fmt.Println(text)
		fmt.Println("")
		fmt.Println(name)
		fmt.Println("~~~~~~ END ~~~~~~")

		conn.Write([]byte{0x1B, 0x40}) // Initialize

		conn.Write([]byte("~~~~~~ MESSAGE ~~~~~~\n"))
		conn.Write([]byte(time.Now().UTC().String()))
		conn.Write([]byte("\n"))
		conn.Write([]byte("\n"))
		conn.Write([]byte(text))
		conn.Write([]byte("\n"))
		conn.Write([]byte(name))
		conn.Write([]byte("\n"))
		conn.Write([]byte("~~~~~~ END ~~~~~~\n"))
	}()

	return ctx.Redirect("/", 302)
}
