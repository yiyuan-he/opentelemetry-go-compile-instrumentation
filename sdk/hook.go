// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	_ "unsafe"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// setupTracing initializes an OTLP exporter and configures the corresponding trace provider
func setupTracing(ctx context.Context) (func(context.Context) error, error) {
	// Create a resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("instrumented-app"),
			semconv.ServiceVersionKey.String("0.1.0"),
			),
		)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Configure stdout trace exporter
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
		)
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
		))

	// Instrument HTTP client
	http.DefaultClient = &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	// Return a function that can be used to clean up resources
	return tracerProvider.Shutdown, nil
}

// Global AWS config with instrumentation
var (
	s3Client    *s3.Client
	awsInitOnce sync.Once
	awsInitErr  error
)

// GetS3Client returns an instrumented S3 client
func GetS3Client() (*s3.Client, error) {
	awsInitOnce.Do(func() {
		// Load the AWS SDK configuration
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			awsInitErr = fmt.Errorf("failed to load AWS SDK configuration: %w", err)
			return
		}

		// Instrument AWS SDK
		otelaws.AppendMiddlewares(&cfg.APIOptions)
		
		// Create S3 client with the instrumented config
		s3Client = s3.NewFromConfig(cfg)
	})

	if awsInitErr != nil {
		return nil, awsInitErr
	}
	
	return s3Client, nil
}

//go:linkname MyHook main.Hook
func MyHook() {
	fmt.Println("Initializing OpenTelemetry instrumentation...")

	// Create context for setup
	ctx := context.Background()

	// Initialize tracing
	shutdown, err := setupTracing(ctx)
	if err != nil {
		fmt.Printf("Error setting up tracing: %v\n", err)
		return
	}

	// Initialize AWS SDK with instrumentation
	_, err = GetS3Client()
	if err != nil {
		fmt.Printf("Error setting up AWS SDK instrumentation: %v\n", err)
		// Continue execution even if AWS setup fails
	} else {
		fmt.Println("AWS SDK instrumentation initialized")
	}

	// Make sure to shut down cleanly
	go func() {
		<-ctx.Done()
		if err := shutdown(context.Background()); err != nil {
			fmt.Printf("Error shutting down tracing: %v\n", err)
		}
	}()

	fmt.Println("OpenTelemetry instrumentation initialized")
}