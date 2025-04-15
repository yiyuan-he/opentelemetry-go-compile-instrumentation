// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
	_ "unsafe"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/open-telemetry/opentelemetry-go-compile-instrumentation/sdk"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Global S3 client for use in s3buckets.go
var s3Client *s3.Client

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() (err error) {
	// Handle SIGINT (CTRL+C) gracefully
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// The OpenTelemetry setup is handled by the injected hook code
	fmt.Println("Starting HTTP server with OpenTelemetry instrumentation...")
	
	// Initialize S3 client
	s3Client, err = sdk.GetS3Client()
	if err != nil {
		fmt.Printf("Error setting up S3 client: %v\n", err)
		// Continue even if S3 setup fails
	}

	// Start HTTP server
	srv := &http.Server{
		Addr:         ":8080",
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(),
	}
	srvErr := make(chan error, 1)
	go func() {
		fmt.Println("Server listening on http://localhost:8080")
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption
	select {
	case err = <-srvErr:
		// Error when starting HTTP server
		return
	case <-ctx.Done():
		// Wait for first CTRL+C
		// Stop receiving signal notifications as soon as possible
		stop()
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed
	fmt.Println("Shutting down server...")
	err = srv.Shutdown(context.Background())
	return
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// handleFunc is a replacement for mux.HandleFunc
	// which enriches the handler's HTTP instrumentation with the pattern as the http.route
	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		// Configure the "http.route" for the HTTP instrumentation
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}

	// Register handlers
	handleFunc("/rolldice", rolldice)
	handleFunc("/rolldice/{player}", rolldice)
	handleFunc("/s3/buckets", listS3Buckets)
	handleFunc("/users", queryUsers)

	// Add HTTP instrumentation for the whole server
	handler := otelhttp.NewHandler(mux, "/")
	return handler
}
