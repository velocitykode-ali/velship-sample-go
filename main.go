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
	// Velocity-style subcommands so the platform's deploy steps (which run
	// `app migrate` for velocity app types before starting `app serve`) are
	// satisfied. Anything other than an explicit no-op command falls through
	// to serving, so a bare `./app` also works for the go-binary app type.
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "", "serve":
		// The web process. `serve` is the velocity app-type start command;
		// a bare `./app` is the go-binary start command. Both serve.
		serve()
	case "velship:processes":
		// The deployer runs this to auto-detect a multi-process manifest.
		// This sample is a single web process, so emit nothing and exit 0 -
		// the deployer falls back to the single web spec. (Serving here would
		// hang the detection step forever and wedge the deploy.)
		return
	default:
		// migrate / queue:work / schedule:work and any other subcommand: no
		// schema, no workers in this sample. Exit 0 so the deploy's migration
		// step succeeds instead of failing a no-DB app.
		fmt.Printf("%s: nothing to do\n", cmd)
		return
	}
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
