package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func listS3Buckets(w http.ResponseWriter, r *http.Request) {
	// Load AWS SDK configuration from environment variables/credentials
	cfg, err := config.LoadDefaultConfig(r.Context())
	if err != nil {
			http.Error(w, "Failed to load AWS configuration: "+err.Error(), http.StatusInternalServerError)
			log.Printf("AWS config error: %v\n", err)
			return
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// List buckets
	result, err := client.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	if err != nil {
			http.Error(w, "Failed to list buckets: "+err.Error(), http.StatusInternalServerError)
			log.Printf("S3 ListBuckets error: %v\n", err)
			return
	}

	// Format response as JSON
	w.Header().Set("Content-Type", "application/json")
	response := make(map[string]interface{})
	bucketNames := make([]string, 0, len(result.Buckets))

	for _, bucket := range result.Buckets {
			if bucket.Name != nil {
					bucketNames = append(bucketNames, *bucket.Name)
			}
	}

	response["buckets"] = bucketNames

	if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Response encoding error: %v\n", err)
	}
}
