package pomdb

type Model struct {
	ID        ObjectID `json:"id"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
	DeletedAt int64    `json:"deleted_at"`
}
