package pomdb

import "time"

type Timestamp time.Time

type Model struct {
	ID        ObjectID  `json:"id"`
	CreatedAt Timestamp `json:"created_at"`
	UpdatedAt Timestamp `json:"updated_at"`
	DeletedAt Timestamp `json:"deleted_at"`
}
