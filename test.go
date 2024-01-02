package pomdb

import (
	"log"
)

type User struct {
	Model
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

var client = Client{
	Bucket: "my-bucket",
	Region: "us-east-1",
}

func Test() {
	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	user := User{
		Name:  "John Doe",
		Email: "john.doe@foo.com",
	}

	if err := client.Create(&user); err != nil {
		log.Fatal(err)
	}

	log.Printf("Created user %s at %d", user.ID, user.CreatedAt)
}
