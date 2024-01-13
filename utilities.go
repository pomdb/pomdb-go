package pomdb

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"reflect"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
)

var pluralizer = pluralize.NewClient()

// getCollectionName returns the name of the collection for the given model,
// which is the plural form of the model's name in snake case.
func getCollectionName(i interface{}) string {
	// Get the type of i, dereferencing if it's a pointer
	typ := reflect.TypeOf(i)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Convert the name to snake case and pluralize it
	name := pluralizer.Plural(strcase.ToSnake(typ.Name()))

	// Log the original and final names
	log.Printf("GetCollectionName: %s -> %s", typ.Name(), name)

	// Return the pluralized, snake_case name
	return name
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
