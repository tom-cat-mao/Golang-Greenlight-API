package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// version represents the application version number. This constant is used to track
// the current version of the API, which can be useful for debugging and monitoring.
const version = "1.0.0"

// config defines the application's runtime configuration settings.
// It encapsulates all configurable parameters needed to run the service,
// including network, environment, and database configurations.
// Fields:
//   - port: Network port the HTTP server listens on (e.g. 4000)
//   - env: Deployment environment ("development", "staging", "production")
//     Determines operational behaviors like logging verbosity
//   - db: Database connection pool configuration:
//   - dsn: Data Source Name for PostgreSQL connection
//   - maxOpenConns: Maximum number of open connections
//   - maxIdleConns: Maximum number of idle connections
//   - maxIdleTime: Maximum duration a connection can remain idle
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
}

// application holds the dependencies for our HTTP handlers, helpers, and middleware.
// This struct is used to group all the application-level dependencies together,
// making them easily accessible to all parts of the application.
//   - config: The application configuration settings.
//   - logger: The structured logger for recording application events and errors.
type application struct {
	config config
	logger *slog.Logger
}

// main is the entry point of the application. It initializes the application,
// sets up the database connection, configures the HTTP server, and starts listening
// for incoming requests.
func main() {
	var cfg config

	// Define and parse command-line flags to configure the application. These flags provide
	// runtime configuration options that can be set when starting the service.
	// Available flags:
	//   - port: HTTP server port (default: 4000)
	//   - env: Runtime environment ("development", "staging", "production")
	//   - db-dsn: PostgreSQL connection string (default: GREENLIGHT_DB_DSN env var)
	//   - db-max-open-conns: Maximum open DB connections (default: 25)
	//   - db-max-idle-conns: Maximum idle DB connections (default: 25)
	//   - db-max-idle-time: Maximum idle time for DB connections (default: 15m)
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	// Add sslmode=disable to the DSN.
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")
	flag.Parse()

	// Create a structured logger that writes log entries to standard output.
	// This logger is used throughout the application to record events and errors.
	// The NewTextHandler is used to format the log output as plain text.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Open a database connection pool. This establishes a connection to the
	// PostgreSQL database using the provided configuration. The connection pool
	// allows for efficient reuse of database connections.
	db, err := openDB(cfg)
	if err != nil {
		// If there's an error connecting to the database, log the error and exit.
		logger.Error("database connection error", "error", err)
		os.Exit(1)
	}

	// Close the database connection pool when the main function exits. This ensures
	// that all database connections are properly closed, releasing resources.
	defer db.Close()

	// Log a message indicating that the database connection pool has been established.
	logger.Info("database connection pool established")

	// Initialize the application struct. This creates an instance of the application
	// struct, passing in the configuration and logger.
	app := &application{
		config: cfg,
		logger: logger,
	}

	// Configure the HTTP server. This sets up the server's address, handler,
	// timeouts, and error logging.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// Start the HTTP server. This begins listening for incoming requests on the
	// configured port.
	logger.Info("starting server", "addr", srv.Addr, "env", cfg.env)

	// If there's an error starting the server, log the error and exit.
	err = srv.ListenAndServe()
	logger.Error("server error", "error", err)
	os.Exit(1)
}

// openDB creates and configures a PostgreSQL database connection pool using the provided configuration.
// It validates the connection by:
// 1. Opening a connection pool with the configured DSN
// 2. Setting connection pool parameters (max open/idle connections, idle timeout)
// 3. Performing a health check via PingContext with a 5-second timeout
// Returns the initialized pool or an error if any step fails.
func openDB(cfg config) (*sql.DB, error) {
	// sql.Open() does not establish any connections to the database.
	// It only validates the DSN and prepares the database connection pool.
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set the maximum number of open connections to the database.
	// This limits the total number of connections that can be established.
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Set the maximum number of idle connections in the pool.
	// These are connections kept ready for immediate reuse.
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Set the maximum time an idle connection can remain in the pool before being closed.
	// This helps prevent stale connections from accumulating.
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	// Create a context with a 5-second timeout. This ensures that the database ping operation
	// will not hang indefinitely if the database is unresponsive.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping the database to check the connection. This sends a simple query to the database
	// to verify that the connection is alive and the database is accessible.
	// If the ping fails, it indicates a problem with the database connection.
	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	// If the ping is successful, the function returns the database connection pool.
	return db, nil
}
