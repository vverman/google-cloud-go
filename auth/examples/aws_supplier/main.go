package main

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/auth/credentials/externalaccount"
	"cloud.google.com/go/auth/oauth2adapt"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// CustomAwsSupplier implements externalaccount.AwsSecurityCredentialsProvider.
//
// In a production environment, you would typically use the official AWS SDK for Go
// (e.g., github.com/aws/aws-sdk-go-v2/config) to resolve these credentials and
// region automatically from various sources (env vars, shared config, EC2 IMDS, etc.).
type CustomAwsSupplier struct{}

// AwsRegion resolves the AWS region.
func (s *CustomAwsSupplier) AwsRegion(ctx context.Context, opts *externalaccount.RequestOptions) (string, error) {
	// Example: simplistic resolution from standard environment variables.
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region, nil
	}
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region, nil
	}
	return "", fmt.Errorf("CustomAwsSupplier: Unable to resolve AWS region from AWS_REGION or AWS_DEFAULT_REGION")
}

// AwsSecurityCredentials retrieves AWS security credentials.
func (s *CustomAwsSupplier) AwsSecurityCredentials(ctx context.Context, opts *externalaccount.RequestOptions) (*externalaccount.AwsSecurityCredentials, error) {
	// Example: simplistic resolution from standard environment variables.
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	if accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("CustomAwsSupplier: Unable to resolve AWS credentials. Ensure AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are set")
	}

	return &externalaccount.AwsSecurityCredentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken, // Optional, used for temporary credentials
	}, nil
}

func main() {
	// This example shows how to use a custom AWS provider to list GCS buckets.
	ctx := context.Background()

	gcpAudience := os.Getenv("GCP_WORKLOAD_AUDIENCE")
	saImpersonationURL := os.Getenv("GCP_SERVICE_ACCOUNT_IMPERSONATION_URL")

	if gcpAudience == "" || saImpersonationURL == "" {
		fmt.Println("Skipping example; required environment variables not set.")
		return
	}

	// 1. Instantiate the custom supplier.
	customSupplier := &CustomAwsSupplier{}

	// 2. Configure the credentials options.
	opts := &externalaccount.Options{
		Audience:                       gcpAudience,
		SubjectTokenType:               "urn:ietf:params:aws:token-type:aws4_request",
		ServiceAccountImpersonationURL: saImpersonationURL,
		AwsSecurityCredentialsProvider: customSupplier,
		Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
	}

	// 3. Create the credentials.
	creds, err := externalaccount.NewCredentials(opts)
	if err != nil {
		fmt.Printf("Failed to create credentials: %v\n", err)
		return
	}

	oauth2Creds := oauth2adapt.Oauth2CredentialsFromAuthCredentials(creds)

	// 4. Use the credentials with a Google Cloud client library (e.g., Storage).
	storageClient, err := storage.NewClient(ctx, option.WithCredentials(oauth2Creds))
	if err != nil {
		fmt.Printf("Failed to create storage client: %v\n", err)
		return
	}
	defer storageClient.Close()

	// Example: List buckets to verify authentication.
	it := storageClient.Buckets(ctx, os.Getenv("GOOGLE_CLOUD_PROJECT"))
	for {
		bkt, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Printf("Failed to list buckets: %v\n", err)
			return
		}
		fmt.Printf("Bucket: %s\n", bkt.Name)
	}
}