package pomdb

type Collection struct {
	Client *Client
	Schema *Schema
}

func (s *Collection) Create(model interface{}) error {
	return nil
}
