module greenlight.tomcat.net

go 1.24.1

require github.com/julienschmidt/httprouter v1.3.0

require github.com/lib/pq v1.10.9

require golang.org/x/time v0.11.0

require (
	github.com/wneessen/go-mail v0.6.2
	golang.org/x/crypto v0.37.0
)

require (
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/go-redis/redis v6.15.9+incompatible // indirect
	github.com/redis/go-redis v6.15.9+incompatible // indirect
	golang.org/x/exp/typeparams v0.0.0-20231108232855-2478ac86f678 // indirect
	golang.org/x/mod v0.23.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	golang.org/x/tools v0.30.0 // indirect
	honnef.co/go/tools v0.6.1 // indirect
)

tool honnef.co/go/tools/cmd/staticcheck
