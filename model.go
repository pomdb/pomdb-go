package pomdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/oklog/ulid/v2"
)

var managedTags = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
	"deleted_at": true,
}

type Model struct {
	ID        ULID      `json:"id" pomdb:"id"`
	CreatedAt Timestamp `json:"created_at" pomdb:"created_at"`
	UpdatedAt Timestamp `json:"updated_at" pomdb:"updated_at"`
	DeletedAt Timestamp `json:"deleted_at" pomdb:"deleted_at"`
}

// ErrInvalidHex indicates that a hex string cannot be converted to an ObjectID.
var ErrInvalidHex = errors.New("[Error] Model: the provided hex string is not a valid ULID")

// ObjectID is the BSON ObjectID type represented as a 12-byte array.
type ULID ulid.ULID

// NewULID generates a new ObjectID.
func NewULID() ULID {
	return ULID(ulid.Make())
}

func (id ULID) String() string {
	return ulid.ULID(id).String()
}

// MarshalJSON customizes the JSON representation of ULID.
func (id ULID) MarshalJSON() ([]byte, error) {
	return json.Marshal(ulid.ULID(id).String())
}

// UnmarshalJSON populates the ULID from a JSON representation.
func (id *ULID) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	ul, err := ulid.Parse(str)
	if err != nil {
		return err
	}

	*id = ULID(ul)
	return nil
}

type Timestamp time.Time

func NewTimestamp() Timestamp {
	return Timestamp(time.Now())
}

func NilTimestamp() Timestamp {
	return Timestamp(time.Time{})
}

// MarshalJSON customizes the JSON representation of Timestamp.
func (ts Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(ts).Unix())
}

// UnmarshalJSON populates the Timestamp from a JSON representation.
func (ts *Timestamp) UnmarshalJSON(b []byte) error {
	var unix int64
	if err := json.Unmarshal(b, &unix); err != nil {
		return err
	}

	*ts = Timestamp(time.Unix(unix, 0))
	return nil
}

// UnmarshalText populates the Timestamp from a text representation.
func (ts *Timestamp) UnmarshalText(b []byte) error {
	i, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return err
	}

	*ts = Timestamp(time.Unix(i, 0))
	return nil
}

// String returns the string representation of the Timestamp.
func (t Timestamp) String() string {
	return fmt.Sprintf("%d", time.Time(t).Unix())
}

// IsNil returns true if the Timestamp is the zero value.
func (t Timestamp) IsNil() bool {
	return time.Time(t).IsZero()
}

func (t Timestamp) After(u Timestamp) bool {
	return time.Time(t).After(time.Time(u))
}

func (t Timestamp) Before(u Timestamp) bool {
	return time.Time(t).Before(time.Time(u))
}

func (t Timestamp) Equal(u Timestamp) bool {
	return time.Time(t).Equal(time.Time(u))
}
