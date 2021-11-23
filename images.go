package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/api/option"

	"cloud.google.com/go/firestore"
)

type Credentials struct {
	UserId                    string `json:"userid"`
	ClientResourceStorageName string `json:"clientresourcestoragename"`
	BucketName                string `json:"bucketname"`
	FolderPath                string `json:"folderpath"`
}

const (
	ProjectID      string = "chat-app-5b482"
	CollectionName string = "UserCred"
	CredentialPath string = "/home/dhananjay/Desktop/firebase-key.json"
)

func Save(cred *Credentials) (*Credentials, error) {
	ctx := context.Background()
	sa := option.WithCredentialsFile(CredentialPath)
	client, err := firestore.NewClient(ctx, ProjectID, sa)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	defer client.Close()

	O, err := client.Collection(CollectionName).Doc(cred.UserId).Set(ctx, cred)
	fmt.Println(O)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	return cred, nil
}

func GetCredentials(userInfo string) (*Credentials, error) {
	ctx := context.Background()
	sa := option.WithCredentialsFile(CredentialPath)
	client, err := firestore.NewClient(ctx, ProjectID, sa)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	defer client.Close()
	firestoreData, err := client.Collection(CollectionName).Doc(userId).Get(ctx)
	if err != nil {
		return nil, err
	}
	cred := &Credentials{
		UserId:                    firestoreData.Data()["UserId"].(string),
		ClientResourceStorageName: firestoreData.Data()["ClientResourceStorageName"].(string),
		BucketName:                firestoreData.Data()["BucketName"].(string),
		FolderPath:                firestoreData.Data()["FolderPath"].(string),
	}
	return cred, nil
}
