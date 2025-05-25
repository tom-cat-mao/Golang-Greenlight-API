package main

import (
	"flag"
	"log"
	"net/http"
)

// Define the HTML content to be served.
const html = `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
</head>
<body>
	<h1>Simple CORS</h1>
	<div id="output"></div>
	<script>
		document.addEventListener('DOMContentLoaded', function() {
			fetch("http://localhost:4000/v1/healthcheck").then(function (response) {
				response.text().then(function(text) {
						document.getElementById("output").innerHTML = text;
					});
				},
				function (err) {
					document.getElementById("output").innerHTML = err;
				}
			);
		});
	</script>
</body>
</html>
`

func main() {
	// Define a command-line flag for the server address.
	addr := flag.String("addr", ":9000", "Server address")
	// Parse the command-line flags.
	flag.Parse()

	// Log the server start message.
	log.Printf("starting server on %s", *addr)

	// Start the HTTP server.
	err := http.ListenAndServe(*addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write the HTML content to the response.
		w.Write([]byte(html))
	}))

	// Log any fatal errors that occur during server startup.
	log.Fatal(err)
}
