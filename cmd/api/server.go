package main

import (
	"context"
	"errors"
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

	// Create a channel to receive errors from the shutdown goroutine.
	shutdownError := make(chan error)

	go func() {
		// Create a channel to receive OS signals (buffered to 1 to avoid missing signals).
		quit := make(chan os.Signal, 1)

		// Notify the quit channel on receiving SIGINT or SIGTERM signals.
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Block until a signal is received.
		s := <-quit

		// Log that the server is shutting down, including the received signal.
		app.logger.Info("shutting down server", "signal", s.String())

		// Create a context with a 30-second timeout for the shutdown process.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel() // Ensure resources are cleaned up.

		// Call Shutdown() on the server as usual,
		// but only send on the shutdownError cahnnel if it returns an error
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		// Log a message to say that we're waiting for any background goroutines to complete their tasks
		app.logger.Info("completing background tasks", "addr", srv.Addr)

		// Call Wait() to block until our WaiGroup counter is zero -- essentially blocking until the background goroutines have finished.
		// Then we return nil on the shurdownError channel, to indicate
		// that the shutdown completed without any issues
		app.wg.Wait()
		shutdownError <- nil
	}()

	// Log that the server is starting, including the address and environment.
	app.logger.Info("starting server", "addr", srv.Addr, "env", app.config.env)

	// Start the HTTP server. This will block until the server is stopped or an error occurs.
	err := srv.ListenAndServe()
	// If the error is not http.ErrServerClosed, it means the server stopped unexpectedly.
	if !errors.Is(err, http.ErrServerClosed) {
		// Return the error to be handled by the caller.
		return err
	}

	// Wait for the shutdown goroutine to finish and receive any error from the shutdown process.
	err = <-shutdownError
	if err != nil {
		// If an error occurred during shutdown, return it to be handled by the caller.
		return err
	}

	app.logger.Info("stopped server", "addr", srv.Addr)

	return nil
}
