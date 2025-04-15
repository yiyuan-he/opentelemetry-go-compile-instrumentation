package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func listS3Buckets(w http.ResponseWriter, r *http.Request) {
	// Use the global S3 client
	if s3Client == nil {
		http.Error(w, "S3 client not initialized", http.StatusInternalServerError)
		log.Println("S3 client not initialized")
		return
	}

	// List buckets using request context for proper trace propagation
	result, err := s3Client.ListBuckets(r.Context(), &s3.ListBucketsInput{})
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
