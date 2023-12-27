# nodb-go

NoDB is an innovative approach to database management, leveraging the robust storage capabilities of [S3]() to store and retrieve data without the need for a traditional database.

> **Table of Contents**
>
> - [Installation](#installation)
> - [Creating a Client](#creating-a-client)
> - [Creating a Schema](#creating-a-schema)
>   - [Object Identifiers](#object-identifiers)
>   - [Generated Fields](#generated-fields)
>   - [Field Validation](#field-validation)
> - [Creating a Collection](#creating-a-collection)
> - [Working with Objects](#working-with-objects)
>   - [Creating](#creating)
>   - [Updating](#updating)
>   - [Deleting](#deleting)
>   - [Querying](#querying)
>     - [Find One](#find-one)
>     - [Find Many](#find-many)
>     - [Find By ID](#find-by-id)
>     - [Find All](#find-all)

## Installation

```bash
go get github.com/nallenscott/nodb-go
```

## Creating a Client

The client is used to manage the location and structure of the database. NoDB requires a dedicated bucket to store data, and the bucket must exist before the client is created.

```go
import (
    "log"

    "github.com/nallenscott/nodb-go"
)

var client = nodb.NewClient(nodb.Client{
    Bucket: "my-bucket",
    Region: "us-east-1",
})

func main() {
    if err := client.Connect(); err != nil {
        log.Fatal(err)
    }

    // ...
}
```

## Creating a Schema

Schemas are used to manage the structure of collections. Schemas are defined using structs, with `json` tags to define the name of the field in the database object.

```go
type Users struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

### Object Identifiers

NoDB automatically generates a unique ID for each object stored in the database. IDs are stored in the `uuid` field of the object in [UUID v4](https://www.ietf.org/archive/id/draft-ietf-uuidrev-rfc4122bis-10.html#name-uuid-version-4) format. There is no need to define this field in the schema.

### Generated Fields

NoDB can automatically generate values for certain types of fields. To enable this, add the `nodb` tag to the field, and set the value to the type of generator to use. The following types are supported:

- `unix` - Generates a Unix timestamp in milliseconds

```go
type Users struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Created  int64  `json:"created" nodb:"unix"`
    Updated  int64  `json:"updated" nodb:"unix"`
}
```

### Field Validation

NoDB will validate the schema of each object before storing it in the database. If the object doesn't match the schema, an error will be returned. NoDB uses [go-playground/validator](https://github.com/go-playground/validator) for schema validation, and supports all of the tags defined by that package.

```go
type Users struct {
    Name     string `json:"name" validate:"required"`
    Email    string `json:"email" validate:"required,email"`
    Created  int64  `json:"created" nodb:"unix"`
    Updated  int64  `json:"updated" nodb:"unix"`
}
```

## Creating a Collection

Collections are groups of objects that share the same schema. If the collection doesn't exist, it will be created. If the schema doesn't match the existing collection, an error will be returned.

```go
users := client.Collection(nodb.Collection{
    Name:   "users",
    Schema: Users{},
})

if err := users.Commit(); err != nil {
    log.Fatal(err)
}

// ...
```

## Working with Objects

Objects are stored in collections, and represent a single record in the database. Objects can be found in S3 using the following path:

```
<bucket>/<collection>/<uuid>.json
```

### Creating

```go
user := users.Create(Users{
    Name:  "John Doe",
    Email: "john.doe@foo.com",
})

if err := user.Commit(); err != nil {
    log.Fatal(err)
}

// ...
```

### Updating

```go
user := users.FindByID("85886d97-ec40-4a56-8569-ff2ea118a2a1")

user := users.Update(user, Users{
    Name:  "Jane Doe",
    Email: "jane.doe@bar.com",
})

if err := user.Commit(); err != nil {
    log.Fatal(err)
}

// ...
```

### Deleting

```go
user := users.FindByID("85886d97-ec40-4a56-8569-ff2ea118a2a1")

if err := user.Delete(); err != nil {
    log.Fatal(err)
}

// ...
```

### Querying

#### <u>Find One</u>

```go
query := users.FindOne(nodb.Query{
    Field: "email",
    Value: "jane.doe@bar.com",
})

if err := query.Execute(); err != nil {
    log.Fatal(err)
}

user := query.Result().(Users)

// ...
```

#### <u>Find Many</u>

```go
query := users.FindMany(nodb.Query{
    Field: "name",
    Value: "Doe",
})

if err := query.Execute(); err != nil {
    log.Fatal(err)
}

users := query.Result().([]Users)

// ...
```

#### <u>Find By ID</u>

```go
user := users.FindByID("85886d97-ec40-4a56-8569-ff2ea118a2a1")

if err := user.Fetch(); err != nil {
    log.Fatal(err)
}

// ...
```

#### <u>Find All</u>

```go
query := users.FindAll()

if err := query.Execute(); err != nil {
    log.Fatal(err)
}

users := query.Result().([]Users)

// ...
```
