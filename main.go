package main

import (
	"errors"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func vcsFromBuildInfo() (repoURL, commit string, modified bool) {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				commit = s.Value
			case "vcs.modified":
				modified = (s.Value == "true")
			}
		}
		mp := info.Main.Path
		if strings.HasPrefix(mp, "github.com/") ||
			strings.HasPrefix(mp, "gitlab.com/") ||
			strings.HasPrefix(mp, "bitbucket.org/") {
			repoURL = "https://" + mp
		}
	}
	return
}

func main() {
	svc := os.Getenv("DD_SERVICE")
	if svc == "" {
		svc = "go-apm-min"
	}
	env := os.Getenv("DD_ENV")

	repoURL, commit, modified := vcsFromBuildInfo()
	log.Printf("BuildVCS: repo=%s commit=%s modified=%v", repoURL, commit, modified)

	// Start the Datadog tracer (defaults to sending to local agent at 127.0.0.1:8126).
	tracer.Start(
		tracer.WithService(svc),
		tracer.WithEnv(env),
	)
	defer tracer.Stop()

	log.Printf("Datadog tracer started (service=%s env=%s). Emitting a span every 30s...", svc, env)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Emit a simple custom span every 30 seconds
	i := 0
	for {
		i++
		span := tracer.StartSpan("heartbeat",
			tracer.ResourceName("heartbeat"),
			tracer.Tag("component", "demo"),
		)
		// Simulate a small bit of work
		time.Sleep(100 * time.Millisecond)

		if i%3 == 0 {
			err := errors.New("simulated failure: database timeout")
			span.SetTag(ext.Error, err)                       // kratko
			span.SetTag("error.type", "TimeoutError")         // tip
			span.SetTag("error.msg", err.Error())             // poruka
			span.SetTag("error.stack", string(debug.Stack())) // stack (bitno za grupisanje)
			span.Finish(tracer.WithError(err))
			log.Println("sent ERROR span")
		} else {
			span.Finish()
			log.Println("sent OK span")
		}
		<-ticker.C
	}
}
