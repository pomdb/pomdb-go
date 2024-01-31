package pomdb

import (
	"encoding/json"
	"time"
)

type Timestamp time.Time

func NewTimestamp() Timestamp {
	return Timestamp(time.Now())
}

// MarshalJSON customizes the JSON representation of Timestamp.
func (ts Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(ts).Unix())
}

// UnmarshalJSON populates the Timestamp from a JSON representation.
func (ts *Timestamp) UnmarshalJSON(b []byte) error {
	var unix int64
	if err := json.Unmarshal(b, &unix); err != nil {
		return err
	}

	*ts = Timestamp(time.Unix(unix, 0))
	return nil
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
