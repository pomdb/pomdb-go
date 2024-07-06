<div>
  <h1 align="center">
    <img src="https://github.com/pomdb/pomdb-go/assets/11765848/6bdacd14-6569-479f-ae53-f02e8b4c2011" width=250 alt=""><br>
    pomdb-go<br>
  </h1>
  <br>
  <p align="center">
    <a href="https://goreportcard.com/report/github.com/pomdb/pomdb-go"><img src="https://goreportcard.com/badge/github.com/pomdb/pomdb-go?style=flat-square"></a>
    <a href="https://pkg.go.dev/github.com/pomdb/pomdb-go"><img src="https://pkg.go.dev/badge/github.com/pomdb/pomdb-go"></a>
    <a href="https://github.com/pomdb/pomdb-go/releases/latest"><img src="https://img.shields.io/github/release/pomdb/pomdb-go.svg?style=flat-square"></a>
  </p>
  <p>
    <strong>PomDB</strong> is a NoSQL object database that leverages the robust storage capabilities of <a href="https://aws.amazon.com/s3">Amazon S3</a> to store and retrieve data. PomDB is entirely client-driven and enforces an opinionated structure for consistency, compatibility, and speed :fire:
  </p>
</div>

## Table of Contents

- [:card_index: Object Databases](#object-databases)
- [:rocket: Feature Highlights](#feature-highlights)
- [:bulb: Use Cases](#use-cases)
- [:package: Installation](#installation)
- [:zap: Quick start](#quick-start)
- [:gear: Creating a Client](#creating-a-client)
- [:hammer: Creating a Model](#creating-a-model)
- [:nut_and_bolt: Working with Objects](#working-with-objects)
- [:mag: Working with Indexes](#working-with-indexes)
- [:page_facing_up: Pagination](#pagination)
- [:balance_scale: Concurrency Control](#concurrency-control)
- [:construction: Roadmap](#roadmap)

## Object Databases

An object database is a type of NoSQL database that stores data as discrete objects, rather than rows and columns. Objects are self-contained units of data that can contain multiple fields, including nested objects and arrays. Object databases are schemaless, meaning that objects can have different fields and data types, and can be updated without changing the database schema.

## Feature Highlights

- Serverless client-driven architecture
- S3-backed [durability]() and [consistency]()
- [Strongly-typed]() and [schemaless]() data storage
- [Pessimistic]() and [optimistic]() concurrency control
- Lexicographically sortable [ULID]() identifiers
- Real-time [change data capture]() via S3 events
- [Soft-deletes]() for reversible data management
- [Inverted indexes]() for fast and efficient querying
- [Pagination]() for large data sets and high throughput
- Have a feature request? [Let us know]()

## Use Cases

- :iphone: Web and Mobile Applications: user data, sessions, application state
- :robot: IoT and Edge Computing: large volumes of sensor data
- :pencil: Content Management Systems: metadata, versioning, content retrieval
- :chart_with_upwards_trend: Data Lakes and Big Data Analytics: vast datasets, analytical insights

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

  res, err := client.Create(&user)
  if err != nil {
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

  "github.com/pomdb/pomdb-go"
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
  FirstName string `json:"first_name" pomdb:"index"`
  LastName  string `json:"last_name" pomdb:"index"`
  Email     string `json:"email" pomdb:"index,unique"`
}
```

> or, defining `ID` field manually

```go
type User struct {
  ID        pomdb.ULID `json:"id" pomdb:"id"`
  FirstName string     `json:"first_name" pomdb:"index"`
  LastName  string     `json:"last_name" pomdb:"index"`
  Email     string     `json:"email" pomdb:"index,unique"`
  //...
}
```

> serializes to:

```json
{
  "id": "01HS8Q7MVGA8CVCVVFYEH1VY2T",
  "first_name": "John",
  "last_name": "Pip",
  "email": "john.pip@zip.com",
  "created_at": 1711210960,
  "updated_at": 1711210960,
  "deleted_at": 0
}
```

### Object Timestamps

Timestamps are used to track when objects are created, updated, and deleted. The native `time.Time` type is used to represent timestamps, and is automatically converted to and from Unix time. Fields with the `created_at`, `updated_at`, and `deleted_at` tags are automatically updated by PomDB:

> embedding `pomdb.Model`

```go
type User struct {
  pomdb.Model
  FirstName string `json:"first_name" pomdb:"index"`
  LastName  string `json:"last_name" pomdb:"index"`
  Email     string `json:"email" pomdb:"index,unique"`
}
```

> or, defining timestamps manually

```go
type User struct {
  ID        pomdb.ULID      `json:"id" pomdb:"id"`
  CreatedAt pomdb.Timestamp `json:"created_at" pomdb:"created_at"`
  UpdatedAt pomdb.Timestamp `json:"updated_at" pomdb:"updated_at"`
  DeletedAt pomdb.Timestamp `json:"deleted_at" pomdb:"deleted_at"`
  FirstName string          `json:"first_name" pomdb:"index"`
  LastName  string          `json:"last_name" pomdb:"index"`
  Email     string          `json:"email" pomdb:"index,unique"`
  //...
}
```

> serializes to:

```json
{
  "id": "01HS8Q7MVGA8CVCVVFYEH1VY2T",
  "first_name": "John",
  "last_name": "Pip",
  "email": "john.pip@zip.com",
  "created_at": 1711210960,
  "updated_at": 1711210960,
  "deleted_at": 0
}
```

## Working with Objects

Objects are stored in collections, and represent a single record in the database. Objects can be found in S3 under the following path:

```hbs
{{$bucket}}/{{$collection}}/{{$ulid}}
```

### Marshalling strategy

PomDB will convert the model name to snake case and pluralize it for the collection name. For example, the `User` model will be stored in the `users` collection. Fields are serialized using the `json` tag, and must be exported. Fields that are not exported will be ignored.

### Query methods

#### `Create(model interface{})`

This method is used to create a new object in the database. `model` must be a pointer to an interface that embeds the `pomdb.Model` struct, or defines an `ID` field of type `pomdb.ULID`, e.g.:

> **Equivalent to** `INSERT INTO users (id, full_name, email) VALUES (...)`

```go
user := User{
  FirstName: "John",
  LastName:  "Pip",
  Email:     "john.pip@zip.com",
}

if err := client.Create(&user); err != nil {
  log.Fatal(err)
}
```

#### `Update(model interface{})`

This method is used to update an existing object in the database. `model` must be a pointer to an interface that embeds the `pomdb.Model` struct, or defines an `ID` field of type `pomdb.ULID`, e.g.:

> **Equivalent to** `UPDATE users SET email = 'jane.pip@zip.com' WHERE id = '...'`

```go
user.Email = "john.pip@zap.com"

if err := client.Update(&user); err != nil {
  log.Fatal(err)
}
```

#### `Delete(model interface{})`

This method is used to delete an existing object in the database. `model` must be a pointer to an interface that embeds the `pomdb.Model` struct, or defines an `ID` field of type `pomdb.ULID`, e.g.:

> **Equivalent to** `DELETE FROM users WHERE id = '...'`

```go
if err := client.Delete(&user); err != nil {
  log.Fatal(err)
}
```

#### `FindOne(query pomdb.Query)`

This method is used to find a single object in the database using an index. The query must include the model, field name, and field value, e.g.:

> **Equivalent to** `SELECT * FROM users WHERE email = 'jane.pip@zip.com'`

```go
query := pomdb.Query{
  Model: User{},
  Field: "email",
  Value: "john.pip@zip.com",
}

res, err := client.FindOne(query)
if err != nil {
  log.Fatal(err)
}

user := res.(*User)
```

#### `FindMany(query pomdb.Query)`

This method is used to find multiple objects in the database using an index. The query must include the model, field name, field value, and filter, e.g.:

> **Equivalent to** `SELECT * FROM users WHERE age < 40`

```go
// Typical HR filter
query := pomdb.Query{
  Model:  User{},
  Field:  "age",
  Filter: pomdb.QueryLessThan,
  Value:  40,
}

res, err := client.FindMany(query)
if err != nil {
  log.Fatal(err)
}

users := make([]User, len(res.Contents))
for i, user := range res.Contents {
  users[i] = user.(User)
}
```

#### `FindAll(query pomdb.Query)`

This method is used to find all objects in the database. The model must be included in the query, e.g.:

> **Equivalent to** `SELECT * FROM users`

```go
query := pomdb.Query{
  Model: User{},
}

res, err := client.FindAll(query)
if err != nil {
  log.Fatal(err)
}

users := make([]User, len(res.Contents))
for i, user := range res.Contents {
  users[i] = user.(User)
}

// ...
```

### Query filters

> [!NOTE]
> [We're working on enhancing query filters with more advanced features →](https://github.com/pomdb/pomdb-go/issues/1)

PomDB provides a basic set of comparison operators for the `Filter` field of the query. If no filter is provided, the query will default to `pomdb.QueryEqual`. Filters may only be used with the [`FindMany`](#findmanyquery-pomdbquery) method. Filters in other query methods are ignored. The list below shows the available filters and their SQL equivalents:

| PomDB Filter             | Equivalent SQL                          |
|:-------------------------|:----------------------------------------|
| `pomdb.QueryEqual`       | `WHERE field = value`                   |
| `pomdb.QueryGreaterThan` | `WHERE field > value`                   |
| `pomdb.QueryLessThan`    | `WHERE field < value`                   |
| `pomdb.QueryBetween`     | `WHERE field BETWEEN value1 AND value2` |

```go
query := pomdb.Query{
  Model:  User{},
  Field:  "age",
  Filter: pomdb.QueryLessThan,
  Value:  40,
}
```

### Soft-deletes

PomDB supports soft-deletes, allowing objects to be marked as deleted without actually removing them from the database. Soft-deleted objects are stored in the database with a non-zero `DeletedAt` object tag, and are automatically excluded from queries. Soft-deleted objects can be restored or purged using the [`Restore`](#restore) and [`Purge`](#purge) methods, respectively. To enable soft-deletes, set the `SoftDeletes` field of the client to `true`:

```go
var client = pomdb.Client{
  Bucket:      "pomdb",
  Region:      "us-east-1",
  SoftDeletes: true,
}
```

#### `Restore(model interface{})`

This method is used to restore a soft-deleted object in the database. `model` must be a pointer to an interface that embeds the `pomdb.Model` struct, or defines an `ID` field of type `pomdb.ULID`, e.g.:

```go
if err := client.Restore(&user); err != nil {
  log.Fatal(err)
}
```

#### `Purge(model interface{})`

This method is used to permanently delete a soft-deleted object and its indexes from the database. `model` must be a pointer to an interface that embeds the `pomdb.Model` struct, or defines an `ID` field of type `pomdb.ULID`, e.g.:

```go
if err := client.Purge(&user); err != nil {
  log.Fatal(err)
}
```

## Working with Indexes

Indexes are used to optimize queries. PomDB supports the following index types, and automatically maintains them when objects are created, updated, or deleted:

### Index types

#### `unique`

Enforces uniqueness of the field's value across the collection. In the example, any `Email` field in `User` structs will be indexed uniquely. PomDB ensures no two `User` records have the same email.

```go
type User struct {
  Email string `pomdb:"index,unique"` // Unique index on Email
  // ...
}
```

> **S3**: `/{{$col}}/indexes/unique/{{$fld}}/{{$val}}/{{$ulid}}`

#### `shared`

Allows multiple records to share the same value for the indexed field. In the example, `Category` is indexed non-uniquely, allowing aggregation and querying of 'Product' records by shared categories.

```go
type Product struct {
  Category string `pomdb:"index"` // Shared index on Category
  // ...
}
```

> **S3**: `/{{$col}}/indexes/shared/{{$fld}}/{{$val}}/{{$[]ulid}}`

#### `ranged`

Facilitates queries within a range of values, like dates or numbers. In the example, `Date` is indexed for ranged queries, allowing for queries like events happening within a certain time frame.

```go
type Event struct {
  Birthday pomdb.Timstamp `pomdb:"index,range"` // Range index on Date
  // ...
}
```

> **S3**: `/{{$col}}/indexes/ranged/{{$fld}}/{{$val}}/{{$[]ulid}}`

### Composite indexes

Composite indexes are used to optimize queries that involve multiple fields. In the example, `IPAddress` and `UserAgent` are indexed together as `IPAddressUserAgent`, allowing for queries that involve both fields. Composite indexes can be unique or shared, and are created by concatenating the field values with a delimiter, e.g. `IPAddress#UserAgent`:

```go
type Log struct {
  IPAddress          string `pomdb:"index,unique"`
  UserAgent          string `pomdb:"index"`
  IPAddressUserAgent string `pomdb:"index,unique"`
  // ...
}

log := Log{
  IPAddress: "172.40.53.24",
  UserAgent: "Mozilla/5.0",
}

log.IPAddressUserAgent = log.IPAddress +"#"+ log.UserAgent

if err := client.Create(&log); err != nil {
  log.Fatal(err)
}

```

### Encoding strategy

PomDB uses base64 encoding to store index values. This allows for a consistent and predictable way to store and retrieve objects, and ensures that the index keys are valid S3 object keys. The length of the index key is limited to 1024 bytes. If the encoded index key exceeds this limit, PomDB will return an error.

## Pagination

PomDB supports pagination using the `Limit` and `NextToken` fields of the query. The `Limit` field is used to specify the maximum number of objects to return per page, and the `NextToken` field is used to specify the starting point for the next page. If there are more objects to return, PomDB will set the `NextToken` field of the response. If there are no more objects to return, `NextToken` will be an empty string:

```go
query := pomdb.Query{
  Model: User{},
  Limit: 10,
}

res, err := client.FindAll(query)
if err != nil {
  log.Fatal(err)
}

for res.NextToken != "" {
  for _, user := range res.Contents {
    // ...
  }

  query.NextToken = res.NextToken
  res, err = client.FindAll(query)
  if err != nil {
    log.Fatal(err)
  }
}

// process the last page
for _, user := range res.Contents {
  // ...
}
```

## Concurrency Control

PomDB implements advanced concurrency mechanisms to efficiently manage data integrity in multi-user environments, including both pessimistic and optimistic concurrency control. This dual approach allows PomDB to cater to a wide range of application requirements, balancing data integrity with system performance.

### Pessimistic

Pessimistic concurrency control locks data during transactions to prevent conflicts. While this approach ensures data consistency by preventing concurrent modifications, it can impact performance in high-traffic scenarios. To enable it, set the `Pessimistic` field of the client to `true`:

```go
var client = pomdb.Client{
  Bucket:      "pomdb",
  Region:      "us-east-1",
  Pessimistic: true,
}
```

### Optimistic

Optimistic concurrency control allows concurrent access and resolves conflicts as they occur. This approach offers higher throughput but may lead to increased conflicts and retries in environments with frequent data updates. To enable it, set the `Optimistic` field of the client to `true`:

```go
var client = pomdb.Client{
  Bucket:     "pomdb",
  Region:     "us-east-1",
  Optimistic: true,
}
```

## Roadmap

You can view the roadmap and feature requests on the [GitHub project page](https://github.com/orgs/pomdb/projects/2).
