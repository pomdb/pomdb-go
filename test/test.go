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
		Email:    "john.doe@zip.com",
	}

	crt, err := client.Create(&user)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("created user with ETag: %s", *crt)

	user.FullName = "Jane Doe"
	user.Email = "jane.doe@zip.com"

	upt, err := client.Update(&user)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("updated user with ETag: %s", *upt)
}
