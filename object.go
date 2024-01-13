package pomdb

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"
)

// ErrInvalidHex indicates that a hex string cannot be converted to an ObjectID.
var ErrInvalidHex = errors.New("the provided hex string is not a valid ObjectID")

// NilObjectID is the zero value for ObjectID.
var NilObjectID ObjectID

var objectIDCounter = readRandomUint32()
var processUnique = processUniqueBytes()

// ObjectID is the BSON ObjectID type represented as a 12-byte array.
type ObjectID [12]byte

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
