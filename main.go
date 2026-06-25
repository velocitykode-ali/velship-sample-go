// Command velship-sample-go is a minimal deployable Go service used to
// exercise the velship add-app -> deploy -> on-demand dependency install
// path. It has no external dependencies so the deploy box builds it with a
// plain `go build -o app .` and runs it with `./app serve`, matching the
// velocity app-type's default build/start commands. A real database is not
// required for the install e2e: selecting PostgreSQL at create time is what
// drives the agent to apt-install the engine; this service only needs to
// come up and answer the health check.
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		fmt.Println("usage: app serve")
		os.Exit(2)
	}
	serve()
}

func serve() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	// Health endpoint Caddy probes after deploy to confirm the process is live.
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		// Surface a couple of env vars so a deploy visibly proves the .env
		// the agent wrote (including any DB_* the engine provisioner injected)
		// reached the running process.
		fmt.Fprintf(w, "velship-sample-go up\n")
		fmt.Fprintf(w, "APP_ENV=%s\n", os.Getenv("APP_ENV"))
		fmt.Fprintf(w, "DB_CONNECTION=%s\n", os.Getenv("DB_CONNECTION"))
		fmt.Fprintf(w, "DB_DATABASE=%s\n", os.Getenv("DB_DATABASE"))
		fmt.Fprintf(w, "REDIS_URL=%s\n", os.Getenv("REDIS_URL"))
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Printf("listening on :%s\n", port)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
