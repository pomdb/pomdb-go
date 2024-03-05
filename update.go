package pomdb

// Update updates a record in the database. It takes a pointer to a struct
// containing the fields that have changed, and a pointer to the model struct
// used to unmarshal json from the database. The struct must have a field with
// the `pomdb:"id"` tag. The function will fetch the existing record and check
// if any index fields have changed. If so, it will remove the old index item
// and create a new one. It will then update the changed fields in the record
// and save it to the database. It's not a new record, so it does not need to
// set the managed fields. It's safe to overwrite an existing index item with
// the same key, so we don't need to check if the index item exists before
// creating it. The function returns the etag of the updated record.
