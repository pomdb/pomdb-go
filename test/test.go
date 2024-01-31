package main

import (
	"log"

	"github.com/pomdb/pomdb-go"
)

type User struct {
	pomdb.Model
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email" validate:"required,email" pomdb:"index"`
}

var client = pomdb.Client{
	Bucket: "pomdb",
	Region: "us-east-1",
}

func main() {
	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	user := User{
		FullName: "John Doe",
		Email:    "john.doe@zab.com",
	}

	if err := client.Create(&user); err != nil {
		log.Fatal(err)
	}

	log.Printf("Created user %s at %s", user.ID.String(), user.CreatedAt.String())
}
