package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// The application version number
const version = "1.0.0"

// Config struct to hold all the configuration settings
//   - the port to listen on
//   - the name of the current operationg environment for the application
//     (development, staging, production, etc)
type config struct {
	port int
	env  string
}

// Application struct to hold the dependencies for our
// HTTP handlers, helpers and middleware
type application struct {
	config config
	logger *slog.Logger
}

func main() {
	var cfg config

	// Read command-line flags
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.Parse()

	// Structured logger which writes log entries to the standard out stream
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app := &application{
		config: cfg,
		logger: logger,
	}

	// Declare HTTP server
	// User the httprouter instance returned by app.routes()
	// as the server handler
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// Start the HTTP server
	logger.Info("starting server", "addr", srv.Addr, "env", cfg.env)

	err := srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}
