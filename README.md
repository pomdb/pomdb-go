<div>
  <h1 align="center">
    <img src="https://github.com/pomdb/pomdb-go/assets/11765848/fad1e057-73d1-4e6f-92d8-804865cf11d2" width=420 alt=""><br>
    pomdb-go<br>
  </h1>
  <br>
  <p align="center">
    <a href="https://goreportcard.com/report/github.com/pomdb/pomdb-go"><img src="https://goreportcard.com/badge/github.com/pomdb/pomdb-go?style=flat-square"></a>
    <a href="https://pkg.go.dev/github.com/pomdb/pomdb-go"><img src="https://pkg.go.dev/badge/github.com/pomdb/pomdb-go"></a>
    <a href="https://github.com/pomdb/pomdb-go/releases/latest"><img src="https://img.shields.io/github/release/pomdb/pomdb-go.svg?style=flat-square"></a>
  </p>
  <p>
    <strong>PomDB</strong> is an innovative approach to database management, leveraging the robust storage capabilities of <a href="https://aws.amazon.com/s3">S3</a> to store and retrieve data. PomDB is entirely client-driven and enforces an opinionated structure for consistency and compatibility. Designed to take the <strong>pain</strong> out of scaling your data, with <strong>simplicity</strong> and <strong>performance</strong> in mind.
  </p>
</div>

## Features

- Serverless client-driven architecture
- S3-backed [durability]() and [consistency]()
- [Pessimistic]() and [optimistic]() concurrency control
- Real-time [change data capture]() via S3 events
- Schema [migration]() and [validation]()

## Installation

```bash
go get github.com/pomdb/pomdb-go
```

## Quick start

```go
package main

import (
	"log"

	"github.com/pomdb/pomdb-go"
)

type User struct {
	pomdb.Model
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
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
		Email:    "john.doe@foo.com",
	}

	if err := client.Create(&user); err != nil {
		log.Fatal(err)
	}

	log.Printf("Created user %s at %d", user.ID, user.CreatedAt)
}
```

## Creating a Client

The client is used to manage the location and structure of the database. PomDB requires a dedicated bucket to store data, and the bucket must exist before the client is created.

```go
import (
  "log"

  "github.com/nallenscott/pomdb-go"
)

var client = pomdb.Client{
  Bucket: "pomdb",
  Region: "us-east-1",
}

func main() {
  if err := client.Connect(); err != nil {
    log.Fatal(err)
  }

  // ...
}
```

## Creating a Model

Models are used to manage the structure of collections. Models are defined using structs, with `json` tags to serialize the data.

```go
type User struct {
  pomdb.Model
  FullName string `json:"full_name"`
  Email    string `json:"email"`
}

//...
```

### Object Conventions

PomDB will convert the model name to snake case and pluralize it for the collection name. For example, the `User` model will be stored in the `users` collection. Fields are serialized using the `json` tag, and must be exported.

### Object Identifiers

PomDB automatically generates an ObjectID for each object stored in the database. IDs are stored in the `ID` field of the object in [ObjectId](https://www.mongodb.com/docs/manual/reference/bson-types/#std-label-objectid) format. Models must embed the `pomdb.Model` struct, or define an `ID` field of type `pomdb.ObjectID`:

```go
type User struct {
  pomdb.Model
  FullName string `json:"full_name"`
  Email    string `json:"email"`
}

// OR

type User struct {
  ID       pomdb.ObjectID `json:"id"`
  FullName string         `json:"full_name"`
  Email    string         `json:"email"`
}

//...
```

### Object Timestamps

When embedding the `pomdb.Model` struct, its fields are automatically added to your model. You can choose to omit these fields, or define them manually. If you choose to define them manually, they must use the same names and types as the fields defined by PomDB:

```go
type User struct {
  pomdb.Model
  FullName  string `json:"full_name"`
  Email     string `json:"email"`
}

// OR

type User struct {
  ID        pomdb.ObjectID  `json:"id"`
  FullName  string          `json:"full_name"`
  Email     string          `json:"email"`
  CreatedAt pomdb.Timestamp `json:"created_at"`
  UpdatedAt pomdb.Timestamp `json:"updated_at"`
  DeletedAt pomdb.Timestamp `json:"deleted_at"`
}

//...
```

### Field Validation

PomDB will validate the model before storing it in the database. PomDB uses [go-playground/validator](https://github.com/go-playground/validator) for validation, and supports all of the tags defined by that package:

```go
type User struct {
  pomdb.Model
  FullName string `json:"full_name" validate:"required"`
  Email    string `json:"email" validate:"required,email"`
}
```

## Working with Objects

Objects are stored in collections, and represent a single record in the database. Objects can be found in S3 under the following path:

```
<bucket>/<pluralized_model_name>/<object_id>.json
```

### Creating

```go
user := User{
  Name:  "John Doe",
  Email: "john.doe@foo.com",
}

if err := client.Create(&user); err != nil {
  log.Fatal(err)
}

// ...
```

### Updating

```go
user.Email = "jane.doe@bar.com"

if err := client.Update(&user); err != nil {
  log.Fatal(err)
}
```

### Deleting

```go
if err := client.Delete(&user); err != nil {
  log.Fatal(err)
}
```

### Querying

#### <u>Find One</u>

```go
query := pomdb.Query{
  Field: "email",
  Value: "jane.doe@bar.com",
}

obj, err := client.FindOne("users", query)
if err != nil {
  log.Fatal(err)
}

user := obj.(*User)

// ...
```

#### <u>Find Many</u>

```go
query := pomdb.Query{
  Field: "name",
  Value: "Doe",
  Flags: pomdb.QueryFlagContains,
}

objs, err := client.FindMany("users", query)
if err != nil {
  log.Fatal(err)
}

users := objs.([]User)

// ...
```

#### <u>Find All</u>

```go
objs, err := client.FindAll("users")
if err != nil {
  log.Fatal(err)
}

users := objs.([]User)

// ...
```
