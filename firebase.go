package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func downloadFile(bucket, object string) ([]byte, error) {
	// bucket := "bucket-name"
	// object := "object-name"
	fmt.Println(bucket)
	fmt.Println(object)
	sa := option.WithCredentialsFile("/home/dhananjay/Desktop/firebase-key.json")
	ctx := context.Background()
	client, err := storage.NewClient(ctx, sa)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	fmt.Println(object)
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %v", object, err)
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll: %v", err)
	}
	// fmt.Fprintf(w, "Blob %v downloaded.\n", object)
	return data, nil
}
