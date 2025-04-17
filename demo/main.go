// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"os"
	_ "unsafe"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	
	_ "github.com/open-telemetry/opentelemetry-go-compile-instrumentation/sdk"
)

func initTracer() *sdktrace.TracerProvider {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		fmt.Printf("failed to initialize stdout exporter: %v\n", err)
		os.Exit(1)
	}
	
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(
			sdktrace.NewSimpleSpanProcessor(exporter),
		),
	)
	
	otel.SetTracerProvider(tp)
	return tp
}

func main() {
	fmt.Println("Starting AWS SDK Instrumentation Demo")
	
	tp := initTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	
	// Create context
	ctx := context.Background()
	
	// Load AWS config
	fmt.Println("Loading AWS configuration...")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Printf("Failed to load AWS config: %v\n", err)
		return
	}
	
	// NOTE: Auto-instrumentation should inject code here to call:
	// otelaws.AppendMiddlewares(&cfg.APIOptions)
	
	// Create S3 client
	fmt.Println("Creating S3 client...")
	s3Client := s3.NewFromConfig(cfg)
	
	// List S3 buckets
	fmt.Println("Listing S3 buckets...")
	result, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		fmt.Printf("Failed to list buckets: %v\n", err)
		return
	}
	
	// Print buckets
	fmt.Println("S3 Buckets:")
	for _, bucket := range result.Buckets {
		fmt.Printf("- %s (created: %s)\n", *bucket.Name, bucket.CreationDate)
	}
	
	fmt.Println("Demo completed successfully")
}