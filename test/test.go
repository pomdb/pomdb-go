package main

import (
	"log"

	"github.com/pomdb/pomdb-go"
)

type User struct {
	pomdb.Model
	FullName string `json:"full_name" validate:"required" pomdb:"index"`
	Email    string `json:"email" validate:"required,email" pomdb:"unique"`
	Phone    string `json:"phone" validate:"required,phone" pomdb:"unique"`
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
		FullName: "John Pip",
		Email:    "john.pip@zip.com",
		Phone:    "1234567890",
	}

	crt, err := client.Create(&user)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("created user with ETag: %s", *crt)

	user.FullName = "Jane Doe"
	user.Email = "jane.pip@zip.com"
	user.Phone = "0987654321"

	upt, err := client.Update(&user)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("updated user with ETag: %s", *upt)
}
