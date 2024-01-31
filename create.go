package pomdb

import (
	"fmt"
	"reflect"
)

func (c *Client) Create(i interface{}) error {
	// Ensure 'i' is a pointer and points to a struct
	rv := reflect.ValueOf(i)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("model must be a pointer to a struct")
	}

	// Ensure types of pomdb model fields are correct
	if err := validateModelFields(i); err != nil {
		return err
	}

	co := getCollectionName(i)

	if ifv := getIndexFieldValues(rv); len(ifv) > 0 {
		if err := c.CheckIndexExists(co, ifv); err != nil {
			return err
		}

		if err := c.CreateIndexItem(co, ifv); err != nil {
			return err
		}
	}

	return nil
}
