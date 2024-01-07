package pomdb

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

// ObjectID is a wrapper around a string that represents a UUID without dashes.
type ObjectID struct {
	Value string
}

// NewObjectID generates a new ObjectID based on a UUID without dashes (Simple UUID)
func NewObjectID() ObjectID {
	// Generate a UUID
	uuid := uuid.New().String()

	// Remove dashes
	uuid = strings.Replace(uuid, "-", "", -1)

	return ObjectID{
		Value: uuid,
	}
}

// MarshalJSON customizes the JSON representation of ObjectID.
func (o ObjectID) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

// String method to easily print the ObjectID as a string.
func (o ObjectID) String() string {
	return o.Value
}
