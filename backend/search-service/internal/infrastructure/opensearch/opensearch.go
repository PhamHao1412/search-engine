package opensearch

import (
	"crypto/tls"
	"log"
	"net/http"

	opensearchgo "github.com/opensearch-project/opensearch-go/v2"
)

// Connect establishes connection to OpenSearch cluster
func Connect(url string) (*opensearchgo.Client, error) {
	client, err := opensearchgo.NewClient(opensearchgo.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: []string{url},
	})
	if err != nil {
		return nil, err
	}

	// Simple Ping test to verify connection
	info, err := client.Info()
	if err != nil {
		log.Printf("Warning: Failed to connect to OpenSearch at startup: %v", err)
	} else {
		defer info.Body.Close()
		log.Println("Successfully connected to OpenSearch cluster.")
	}

	return client, nil
}
