package pomdb

import (
	"log"
)

type User struct {
	Id        ObjectId `json:"_id"`
	Name      string   `json:"name" validate:"required"`
	Email     string   `json:"email" validate:"required,email"`
	CreatedAt int64    `json:"created_at,omitempty"`
	UpdatedAt int64    `json:"updated_at,omitempty"`
}

func main() {
	client := Client{
		Bucket: "my-bucket",
		Region: "us-east-1",
	}

	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	schema := Schema{
		Timestamps: true,
	}

	collection := Collection[User]{
		Client: &client,
		Schema: schema,
	}

	user := User{
		Name:  "Alice",
		Email: "alice@example.com",
	}

	model := collection.NewModel(&user)

	if err := model.Save(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Saved user %s", model.Value.Id)

}
