package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// serve starts the HTTP server and handles graceful shutdown on SIGINT or SIGTERM.
func (app *application) serve() error {
	// Create a new http.Server struct with configuration from the application.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),                      // Set the server address using the configured port.
		Handler:      app.routes(),                                             // Set the HTTP handler (router) for incoming requests.
		IdleTimeout:  time.Minute,                                              // Maximum amount of time to wait for the next request.
		ReadTimeout:  5 * time.Second,                                          // Maximum duration for reading the entire request.
		WriteTimeout: 10 * time.Second,                                         // Maximum duration before timing out writes of the response.
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError), // Custom error logger for the server.
	}

	// Start a background goroutine to listen for OS interrupt or terminate signals.
	go func() {
		quit := make(chan os.Signal, 1) // Channel to receive OS signals.

		// Notify the quit channel on SIGINT (Ctrl+C) or SIGTERM (termination).
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		s := <-quit // Block until a signal is received.

		// Log the caught signal and exit the application.
		app.logger.Info("caught signal", "signal", s.String())
		os.Exit(0)
	}()

	// Log that the server is starting, including the address and environment.
	app.logger.Info("starting server", "addr", srv.Addr, "env", app.config.env)

	// Start the HTTP server and return any error encountered.
	return srv.ListenAndServe()
}
