package pomdb

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

var managedTags = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
	"deleted_at": true,
}

type Model struct {
	ID        ObjectID  `json:"id" pomdb:"id"`
	CreatedAt Timestamp `json:"created_at" pomdb:"created_at"`
	UpdatedAt Timestamp `json:"updated_at" pomdb:"updated_at"`
	DeletedAt Timestamp `json:"deleted_at" pomdb:"deleted_at"`
}

// ErrInvalidHex indicates that a hex string cannot be converted to an ObjectID.
var ErrInvalidHex = errors.New("[Error] ObjectID: the provided hex string is not a valid ObjectID")

var objectIDCounter = readRandomUint32()
var processUnique = processUniqueBytes()

// ObjectID is the BSON ObjectID type represented as a 12-byte array.
type ObjectID [12]byte

// String returns the hex encoding of the ObjectID as a string.
func (id ObjectID) String() string {
	return hex.EncodeToString(id[:])
}

// MarshalJSON customizes the JSON representation of ObjectID.
func (id ObjectID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

// UnmarshalJSON populates the ObjectID from a JSON representation.
func (id *ObjectID) UnmarshalJSON(b []byte) error {
	var hexStr string
	if err := json.Unmarshal(b, &hexStr); err != nil {
		return err
	}

	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return err
	}

	if len(decoded) != len(id) {
		return ErrInvalidHex
	}

	copy(id[:], decoded)
	return nil
}

// NewObjectID generates a new ObjectID.
func NewObjectID() ObjectID {
	return NewObjectIDFromTimestamp(time.Now())
}

// NewObjectIDFromTimestamp generates a new ObjectID based on the given time.
func NewObjectIDFromTimestamp(timestamp time.Time) ObjectID {
	var b [12]byte

	binary.BigEndian.PutUint32(b[0:4], uint32(timestamp.Unix()))
	copy(b[4:9], processUnique[:])
	putUint24(b[9:12], atomic.AddUint32(&objectIDCounter, 1))

	return b
}

// processUniqueBytes returns the process unique bytes.
func processUniqueBytes() [5]byte {
	var b [5]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(fmt.Errorf("cannot generate process unique bytes: %v", err))
	}
	return b
}

// readRandomUint32 returns a random uint32.
func readRandomUint32() uint32 {
	var b [4]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(fmt.Errorf("cannot generate random uint32: %v", err))
	}
	return binary.BigEndian.Uint32(b[:])
}

// putUint24 puts a uint32 into a byte slice as a 24-bit big endian value.
func putUint24(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
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

// String returns the string representation of the Timestamp.
func (t Timestamp) String() string {
	mtv, err := time.Time(t).MarshalText()
	if err != nil {
		return ""
	}

	return string(mtv)
}

// IsNil returns true if the Timestamp is the zero value.
func (t Timestamp) IsNil() bool {
	return time.Time(t).IsZero()
}
