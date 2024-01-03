package pomdb

import (
	"crypto/rand"
	"fmt"
)

type ObjectID [12]byte

func NewObjectID() ObjectID {
	var id ObjectID
	rand.Read(id[:]) // Replace with a more robust implementation.
	return id
}

func (id ObjectID) String() string {
	return fmt.Sprintf("%x", id[:])
}
