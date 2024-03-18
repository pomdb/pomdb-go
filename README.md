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
    <strong>PomDB</strong> is an innovative approach to database management, leveraging the robust storage capabilities of <a href="https://aws.amazon.com/s3">S3</a> and <a href="https://min.io">MinIO</a> to store and retrieve data. PomDB is entirely client-driven and enforces an opinionated structure for consistency, compatibility, and speed :fire:
  </p>
</div>

## Features

- Serverless client-driven architecture
- S3-backed [durability]() and [consistency]()
- [Pessimistic]() and [optimistic]() concurrency control
- Real-time [change data capture]() via S3 events
- [Indexes]() for fast and efficient querying
- Have a feature request? [Let us know]()

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
  FullName string `json:"full_name" pomdb:"index"`
  Email    string `json:"email" pomdb:"index,unique"`
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
  }

  if res, err := client.Create(&user); err != nil {
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

Models are used to manage the structure of objects stored in collections. Models are defined using structs, with `json` tags to serialize the data. When embedding the `pomdb.Model` struct, its fields are automatically added to your model. You can choose to omit these fields, or define them manually. If you choose to define them manually, they must use the same names, types, and tags as the fields defined by PomDB:

> embedding `pomdb.Model`
```go
type User struct {
  pomdb.Model
  FullName  string `json:"full_name" pomdb:"index"`
  Email     string `json:"email" pomdb:"index,unique"`
}
```

> or, defining fields manually
```go
type User struct {
  ID        pomdb.ULID      `json:"id" pomdb:"id"`
  CreatedAt pomdb.Timestamp `json:"created_at" pomdb:"created_at"`
  UpdatedAt pomdb.Timestamp `json:"updated_at" pomdb:"updated_at"`
  DeletedAt pomdb.Timestamp `json:"deleted_at" pomdb:"deleted_at"`
  FullName  string          `json:"full_name" pomdb:"index"`
  Email     string          `json:"email" pomdb:"index,unique"`
}
```

### Object Identifiers

PomDB automatically generates a Universally Unique Lexicographically Sortable Identifer ([ULID](https://github.com/ulid/spec?tab=readme-ov-file)) for each object stored in the database. IDs are stored in the `ID` field of the struct, and serialized to the `id` attribute in the json output. Models must embed the `pomdb.Model` struct, or define an `ID` field of type `pomdb.ULID`:

> embedding `pomdb.Model`
```go
type User struct {
  pomdb.Model
  FullName string `json:"full_name" pomdb:"index"`
  Email    string `json:"email" pomdb:"index,unique"`
}
```

> or, defining `ID` field manually
```go
type User struct {
  ID       pomdb.ULID `json:"id" pomdb:"id"`
  FullName string     `json:"full_name" pomdb:"index"`
  Email    string     `json:"email" pomdb:"index,unique"`
  //...
}
```

> model serializes to:
```json
{
  "id": "01HS8Q7MVGA8CVCVVFYEH1VY2T",
  "full_name": "John Pip",
  "email": "john.pip@zip.com",
  "created_at": 1630000000,
  "updated_at": 1630000000,
  "deleted_at": 0
}
```

### Object Timestamps

PomDB timestamps are used to track when objects are created, updated, and deleted. Timestamps are provided by `pomdb.Model` as `CreatedAt`, `UpdatedAt`, and `DeletedAt` fields in `time.Time` format, and are serialized to `created_at`, `updated_at`, and `deleted_at` attributes in Unix seconds format:

> embedding `pomdb.Model`
```go
type User struct {
  pomdb.Model
  FullName  string `json:"full_name" pomdb:"index"`
  Email     string `json:"email" pomdb:"index,unique"`
}
```

> model serializes to:
```json
{
  "id": "01HS8Q7MVGA8CVCVVFYEH1VY2T",
  "full_name": "John Pip",
  "email": "john.pip@zip.com",
  "created_at": 1710765131,
  "updated_at": 1710765131,
  "deleted_at": 0
}
```

## Working with Objects

Objects are stored in collections, and represent a single record in the database. Objects can be found in S3 under the following path:

```
<bucket>/<collection_name>/<object_id>
```

### Marshalling strategy

PomDB will convert the model name to snake case and pluralize it for the collection name. For example, the `User` model will be stored in the `users` collection. Fields are serialized using the `json` tag, and must be exported. Fields that are not exported will be ignored.

### Query methods

#### `Create`

This method is used to create a new object in the database. The object must be a pointer to a struct that embeds the `pomdb.Model` struct, or defines an `ID` field of type `pomdb.ULID`, e.g.:

> **Equivalent to** `INSERT INTO users (id, full_name, email) VALUES (...)`

```go
user := User{
  Name:  "John Pip",
  Email: "john.pip@zip.com",
}

if err := client.Create(&user); err != nil {
  log.Fatal(err)
}

// ...
```

#### `Update`

This method is used to update an existing object in the database. The object must be a pointer to a struct that embeds the `pomdb.Model` struct, or defines an `ID` field of type `pomdb.ULID`, e.g.:

> **Equivalent to** `UPDATE users SET email = 'jane.pip@zip.com' WHERE id = '...'`

```go
user.Email = "jane.pip@zip.com"

if err := client.Update(&user); err != nil {
  log.Fatal(err)
}
```

#### `Delete`

This method is used to delete an existing object in the database. The object must be a pointer to a struct that embeds the `pomdb.Model` struct, or defines an `ID` field of type `pomdb.ULID`, e.g.:

> **Equivalent to** `DELETE FROM users WHERE id = '...'`

```go
if err := client.Delete(&user); err != nil {
  log.Fatal(err)
}
```

#### `FindOne`

This method is used to find a single object in the database using an index. The query must include the model, field name, and field value, e.g.:

> **Equivalent to** `SELECT * FROM users WHERE email = 'jane.pip@zip.com'`

```go
query := pomdb.Query{
  Model:      User{},
  FieldName:  "email",
  FieldValue: "jane.pip@zip.com",
}

obj, err := client.FindOne(query)
if err != nil {
  log.Fatal(err)
}

user := obj.(*User)

// ...
```

#### `FindMany`

This method is used to find multiple objects in the database using an index. The query must include the model, field name, field value, and filter, e.g.:

> **Equivalent to** `SELECT * FROM users WHERE name LIKE '%Doe%'`

```go
query := pomdb.Query{
  Model:      User{},
  FieldName:  "name",
  FieldValue: "Doe",
  Filter:      pomdb.QueryFlagContains,
}

objs, err := client.FindMany(query)
if err != nil {
  log.Fatal(err)
}

users := objs.([]User)

// ...
```

#### `FindAll`

This method is used to find all objects in the database. The model must be included in the query, e.g.:

> **Equivalent to** `SELECT * FROM users`

```go
objs, err := client.FindAll("users")
if err != nil {
  log.Fatal(err)
}

users := objs.([]User)

// ...
```

## Working with Indexes

Indexes are used to optimize queries. PomDB supports unique and non-unique indexes using the `pomdb:"index,unique"` and `pomdb:"index"` tags, respectively, and automatically maintains them when objects are created, updated, or deleted. Indexes can be found in S3 under the following path:

```
<bucket>/<collection_name>/indexes/<field_name>/<base64_encoded_value>
```

### Encoding strategy

PomDB uses base64 encoding to store index values. This allows for a consistent and predictable way to store and retrieve objects, and ensures that the index keys are valid S3 object keys. The length of the index key is limited to 1024 bytes. If the encoded index key exceeds this limit, PomDB will return an error.

### Query filters

#### `QueryFlagEquals`

This is the default filter, and is used to find objects where the field is equal to the specified value, e.g.:

> **Equivalent to** `SELECT * FROM users WHERE email = 'john.pip@zip.com'`

```go
query := pomdb.Query{
  Model:      User{},
  FieldName:  "email",
  FieldValue: "john.pip@zip.com",
  Filter:      pomdb.QueryFlagEquals,
  // ..........^
}
```

#### `QueryFlagContains`

This filter is used to find objects where the field contains the specified value, e.g.:

> **Equivalent to** `SELECT * FROM users WHERE name LIKE '%Pip%'`

```go
query := pomdb.Query{
  Model:      User{},
  FieldName:  "name",
  FieldValue: "Pip",
  Filter:      pomdb.QueryFlagContains,
  // ..........^
}
```

#### `QueryFlagStartsWith`

This filter is used to find objects where the field starts with the specified value, e.g.:

> **Equivalent to** `SELECT * FROM users WHERE name LIKE 'John%'`

```go
query := pomdb.Query{
  Model:      User{},
  FieldName:  "name",
  FieldValue: "John",
  Filter:      pomdb.QueryFlagStartsWith,
  // ..........^
}
```

#### `QueryFlagEndsWith`

This filter is used to find objects where the field ends with the specified value, e.g.:

> **Equivalent to** `SELECT * FROM users WHERE name LIKE '%Pip'`

```go
query := pomdb.Query{
  Model:      User{},
  FieldName:  "name",
  FieldValue: "Pip",
  Filter:      pomdb.QueryFlagEndsWith,
  // ..........^
}
```

#### `QueryFlagGreaterThan`

This filter is used to find objects where the field is greater than the specified value, e.g.:

> **Equivalent to** `SELECT * FROM users WHERE age > 21`

```go
query := pomdb.Query{
  Model:      User{},
  FieldName:  "age",
  FieldValue: 21,
  Filter:      pomdb.QueryFlagGreaterThan,
  // ..........^
}
```

#### `QueryFlagLessThan`

This filter is used to find objects where the field is less than the specified value, e.g.:

> **Equivalent to** `SELECT * FROM users WHERE age < 21`

```go
query := pomdb.Query{
  Model:      User{},
  FieldName:  "age",
  FieldValue: 21,
  Filter:      pomdb.QueryFlagLessThan,
  // ..........^
}
```

