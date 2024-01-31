package pomdb

import "time"

type Timestamp time.Time

func NewTimestamp() Timestamp {
	return Timestamp(time.Now())
}

// String returns the string representation of the Timestamp.
func (t Timestamp) String() string {
	mtv, err := time.Time(t).MarshalText()
	if err != nil {
		return ""
	}

	return string(mtv)
}

type Model struct {
	ID        ObjectID  `json:"id"`
	CreatedAt Timestamp `json:"created_at"`
	UpdatedAt Timestamp `json:"updated_at"`
	DeletedAt Timestamp `json:"deleted_at"`
}
