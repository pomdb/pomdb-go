package main

import (
	"log"

	"github.com/pomdb/pomdb-go"
)

type User struct {
	pomdb.Model
	FullName string `json:"full_name" validate:"required" pomdb:"index"`
	Email    string `json:"email" validate:"required,email" pomdb:"index,unique"`
	Phone    string `json:"phone" validate:"required,phone" pomdb:"index,unique"`
}

var client = pomdb.Client{
	Bucket: "pomdb",
	Region: "us-east-1",
}

func main() {
	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	// user := User{
	// 	FullName: "John Pip",
	// 	Email:    "john.pip@zip.com",
	// 	Phone:    "1234567890",
	// }

	// crt, err := client.Create(&user)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Printf("created user with ETag: %s", *crt)

	// user.FullName = "Jane Doe"
	// user.Email = "jane.pip@zip.com"
	// user.Phone = "0987654321"

	// upt, err := client.Update(&user)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Printf("updated user with ETag: %s", *upt)

	// del, err := client.Delete(&user)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Printf("deleted data for ID: %s", *del)

	// query := pomdb.Query{
	// 	Model: &User{},
	// 	Field: "email",
	// 	Value: "jane.pip@zip.com",
	// }

	// obj, err := client.FindOne(query)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// doc := obj.(*User)

	// log.Printf("FindOne: found user %s with ID %s", doc.FullName, doc.ID)

	query := pomdb.Query{
		Model:  &User{},
		Field:  "full_name",
		Value:  "Doe",
		Filter: pomdb.QueryFilterContains,
	}

	res, err := client.FindMany(query)
	if err != nil {
		log.Fatal(err)
	}

	users := make([]User, len(res.Docs))
	for i, user := range res.Docs {
		users[i] = user.(User)
	}

	log.Printf("FindMany: found %d users", len(users))
}
