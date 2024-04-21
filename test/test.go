package main

import (
	"log"
	"time"

	"github.com/pomdb/pomdb-go"
)

type User struct {
	pomdb.Model
	FirstName string          `json:"first_name" pomdb:"index"`
	LastName  string          `json:"last_name" pomdb:"index"`
	Birthday  pomdb.Timestamp `json:"birthday" pomdb:"ranged,index"`
	Email     string          `json:"email" pomdb:"index,unique"`
	Phone     string          `json:"phone" pomdb:"index,unique"`
	Age       int             `json:"age" pomdb:"index,ranged"`
}

var client = pomdb.Client{
	Bucket:      "pomdb",
	Region:      "us-east-1",
	SoftDeletes: true,
}

func main() {
	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	bday := time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC)

	user := User{
		FirstName: "John",
		LastName:  "Pip",
		Email:     "john.pip@zip.com",
		Phone:     "1234567890",
		Birthday:  pomdb.Timestamp(bday),
		Age:       30,
	}

	crt, err := client.Create(&user)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("created user with ETag: %s", *crt)

	// bday = time.Date(1995, time.January, 1, 0, 0, 0, 0, time.UTC)

	// user.FirstName = "Jane"
	// user.Email = "jane.pip@zip.com"
	// user.Phone = "0987654321"
	// user.Birthday = pomdb.Timestamp(bday)
	// user.Age = 25

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

	// res, err := client.Restore(&user)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Printf("restored data for ID: %s", *res)

	// pur, err := client.Purge(&user)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Printf("purged data for ID: %s", *pur)

	// query := pomdb.Query{
	// 	Model: &User{},
	// 	Field: "email",
	// 	Value: "john.pip@zip.com",
	// }

	// obj, err := client.FindOne(query)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// doc := obj.(*User)

	// log.Printf("FindOne: found user %s with ID %s", doc.FirstName, doc.ID)

	// query := pomdb.Query{
	// 	Model: &User{},
	// 	Field: "first_name",
	// 	Value: "John",
	// }

	// res, err := client.FindMany(query)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// users := make([]*User, len(res.Docs))
	// for i, user := range res.Docs {
	// 	users[i] = user.(*User)
	// }

	// log.Printf("FindMany: found %d users", len(users))

	query := pomdb.Query{
		Model: &User{},
	}

	res, err := client.FindAll(query)
	if err != nil {
		log.Fatal(err)
	}

	users := make([]*User, len(res.Docs))
	for i, user := range res.Docs {
		users[i] = user.(*User)
	}

	log.Printf("FindAll: found %d users", len(users))
}
