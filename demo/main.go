// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"log"
	_ "unsafe"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	_ "github.com/open-telemetry/opentelemetry-go-compile-instrumentation/sdk"
)

func main() {
	fmt.Printf("Hello World\n")
	fmt.Println("Starting AWS SDK demo")
	// Load AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS SDK config: %v", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg)

	// List S3 buckets
	result, err := client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		log.Fatalf("Failed to list buckets: %v", err)
	}

	// Print bucket names
	fmt.Println("S3 Buckets:")
	for _, bucket := range result.Buckets {
		fmt.Println(*bucket.Name)
	}
}
