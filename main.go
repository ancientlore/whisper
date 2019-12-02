package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/facebookgo/flagenv"
)

func main() {
	var (
		fPort = flag.Int("port", 8080, "Port to listen on.")
		fRoot = flag.String("root", ".", "Root of web site.")
	)
	flag.Parse()
	flagenv.Parse()

	var srv = http.Server{
		Addr: fmt.Sprintf(":%d", *fPort),
	}

	http.Handle("/sitemap.txt", gziphandler.GzipHandler(http.HandlerFunc(sitemap)))
	http.Handle("/", http.FileServer(http.Dir(*fRoot))) // temporary

	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)
		// sigterm signal sent from kubernetes
		signal.Notify(sigint, syscall.SIGTERM)

		<-sigint

		// We received an interrupt signal, shut down.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
	}()

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server: %v", err)
	} else {
		log.Print("Goodbye.")
	}
}
