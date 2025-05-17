package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"greenlight.tomcat.net/internal/data"
	"greenlight.tomcat.net/internal/mailer"
)

// version represents the application version number. This constant is used to track
// the current version of the API, which can be useful for debugging and monitoring.
const version = "1.0.0"

// config holds all runtime configuration settings for the application.
// This includes network, environment, database, and rate limiter options.
// Fields:
//   - port: The TCP port for the HTTP server (e.g., 4000).
//   - env: The application environment ("development", "staging", "production").
//   - db: Database connection pool settings, including:
//   - dsn: PostgreSQL Data Source Name.
//   - maxOpenConns: Maximum number of open DB connections.
//   - maxIdleConns: Maximum number of idle DB connections.
//   - maxIdleTime: Maximum time a connection can remain idle.
//   - limiter: Rate limiter configuration, including:
//   - rps: Requests per second allowed.
//   - burst: Maximum burst size for rate limiting.
//   - enabled: Whether rate limiting is enabled.
//   - smtp: the config for smtp server, including:
//   - host: the hostname
//   - port: the post number
//   - username: the user's name
//   - password: the user's password
//   - sender: the sender's name
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

// application represents the core dependencies used throughout the application.
// This struct serves as a centralized container for all application-level components,
// providing easy access to shared resources across handlers, helpers, and middleware.
// Fields:
//   - config: Runtime configuration settings (port, environment, database, etc.)
//   - logger: Structured logger for application events and error reporting
//   - models: Database access layer containing all data operations
//   - mailer: Email sending client struct
//     = wg: sync.WaitGroup to count the goroutine the the background
type application struct {
	config config
	logger *slog.Logger
	models data.Models
	mailer *mailer.Mailer
	wg     sync.WaitGroup
}

// main is the entry point of the application. It initializes the application,
// sets up the database connection, configures the HTTP server, and starts listening
// for incoming requests.
func main() {
	var cfg config

	// Register command-line flag for the API server port (default: 4000)
	flag.IntVar(&cfg.port, "port", 4000, "API server port")

	// Register command-line flag for the application environment (default: "development")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	// Register command-line flag for the PostgreSQL DSN, defaulting to the GREENLIGHT_DB_DSN environment variable
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")

	// Register command-line flag for the maximum number of open database connections (default: 25)
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")

	// Register command-line flag for the maximum number of idle database connections (default: 25)
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")

	// Register command-line flag for the maximum idle time for database connections (default: 15 minutes)
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	// Register command-line flag for the rate limiter's maximum requests per second (default: 2)
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")

	// Register command-line flag for the rate limiter's maximum burst size (default: 4)
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")

	// Register command-line flag to enable or disable the rate limiter (default: true)
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	// Register command-line flag for the smtp server hostname
	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")

	// Register command-line flag for the smport of the smtp server
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")

	// Register command-line flag for the smtp username
	flag.StringVar(&cfg.smtp.username, "smtp-username", "da827255e7cf4c", "SMTP username")

	// Register command-line flag for the smtp password
	flag.StringVar(&cfg.smtp.password, "smtp-password", "c0eb95a13f692e", "SMTP password")

	// Register command-line flag for the smtp sender (default set as my email address)
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "maoy896@gmail.com", "SMTP sender")

	// Parse all registered command-line flags and populate the cfg struct
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

	// Initialize the mailer using the settings from the command line flags
	mailer, err := mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Initialize the application struct. This creates an instance of the application
	// struct, passing in the configuration and logger.
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer,
	}

	// Start the HTTP server and listen for incoming requests.
	// If an error occurs while starting or running the server, log the error and exit the application.
	err = app.serve()
	if err != nil {
		// Log the error message using the structured logger.
		logger.Error(err.Error())
		// Exit the application with a non-zero status code to indicate failure.
		os.Exit(1)
	}
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
