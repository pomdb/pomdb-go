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
go get github.com/nallenscott/pomdb-go
```

## Quick start

```go
package main

import (
  "log"

  "github.com/nallenscott/pomdb-go"
)

type User struct {
  Name     string `json:"name" validate:"required"`
  Email    string `json:"email" validate:"required,email"`
  Created  int64  `json:"created" pomdb:"unix"`
  Updated  int64  `json:"updated" pomdb:"unix"`
}

var client = pomdb.Client{
  Bucket: "my-bucket",
  Region: "us-east-1",
}

func main() {
  if err := client.Connect(); err != nil {
    log.Fatal(err)
  }

  users := &pomdb.Collection[User]{
    Client: client,
    Schema: pomdb.Schema{
      Model: User{},
    },
  }

  user := users.Create(User{
    Name:  "John Doe",
    Email: "john.doe@foo.com",
  })

  if err := user.Save(); err != nil {
    log.Fatal(err)
  }

  log.Printf("Created user %s at %d", user.UUID(), user.Created())
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
  Bucket: "my-bucket",
  Region: "us-east-1",
}

func main() {
  if err := client.Connect(); err != nil {
    log.Fatal(err)
  }

  // ...
}
```

## Creating a Schema

Schemas are used to manage the structure of collections. Schemas are defined using structs, with `json` tags to define the name of the field in the database object. Schemas are optional, you can also store arbitrary data.

```go
type User struct {
  Name  string `json:"name"`
  Email string `json:"email"`
}

schema := pomdb.Schema{
  Model: User{},
}

//...
```

### Object Identifiers

PomDB automatically generates a unique ID for each object stored in the database. IDs are stored in the `uuid` field of the object in [UUID v4](https://www.ietf.org/archive/id/draft-ietf-uuidrev-rfc4122bis-10.html#name-uuid-version-4) format. There is no need to define this field in the schema.

### Generated Fields

PomDB can automatically generate values for certain types of fields. To enable this, add the `pomdb` tag to the field, and set the value to the type of generator to use. The following types are supported:

- `unix` - Generates a Unix timestamp in milliseconds

```go
type User struct {
  Name     string `json:"name"`
  Email    string `json:"email"`
  Created  int64  `json:"created" pomdb:"unix"`
  Updated  int64  `json:"updated" pomdb:"unix"`
}
```

### Field Validation

PomDB will validate the schema of each object before storing it in the database. If the object doesn't match the schema, an error will be returned. PomDB uses [go-playground/validator](https://github.com/go-playground/validator) for schema validation, and supports all of the tags defined by that package.

```go
type User struct {
  Name     string `json:"name" validate:"required"`
  Email    string `json:"email" validate:"required,email"`
  Created  int64  `json:"created" pomdb:"unix"`
  Updated  int64  `json:"updated" pomdb:"unix"`
}
```

## Creating a Collection

Collections are groups of objects that share the same schema. If the collection doesn't exist, it will be created. If the schema doesn't match the existing collection, an error will be returned.

```go
users := pomdb.Collection[User]{
  Client: client,
  Schema: schema,
}

// ...
```

## Working with Objects

Objects are stored in collections, and represent a single record in the database. Objects can be found in S3 under the following path:

```
<bucket>/<collection>/<uuid>.json
```

### Creating

```go
user := users.Create(users.Model{
  Name:  "John Doe",
  Email: "john.doe@foo.com",
})

if err := user.Save(); err != nil {
  log.Fatal(err)
}

// ...
```

### Updating

```go
user.Email = "jane.doe@bar.com"

if err := user.Save(); err != nil {
    log.Fatal(err)
}
```

### Deleting

```go
if err := user.Delete(); err != nil {
    log.Fatal(err)
}
```

### Querying

#### <u>Find One</u>

```go
query := users.FindOne(pomdb.Query{
    Field: "email",
    Value: "jane.doe@bar.com",
})

if err := query.Execute(); err != nil {
    log.Fatal(err)
}

user := query.Result().(User)

// ...
```

#### <u>Find Many</u>

```go
query := users.FindMany(pomdb.Query{
    Field: "name",
    Value: "Doe",
    Flags: pomdb.QueryFlagContains,
})

if err := query.Execute(); err != nil {
    log.Fatal(err)
}

users := query.Result().([]User)

// ...
```

#### <u>Find All</u>

```go
query := users.FindAll()

if err := query.Execute(); err != nil {
    log.Fatal(err)
}

users := query.Result().([]User)

// ...
```
