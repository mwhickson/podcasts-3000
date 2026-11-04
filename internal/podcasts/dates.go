package podcasts

import (
	"fmt"
	"strings"
	"time"
)

type nullableTime struct {
	value *time.Time
}

func (t *nullableTime) Scan(value any) error {
	t.value = nil
	switch value := value.(type) {
	case nil:
		return nil
	case time.Time:
		t.value = &value
		return nil
	case string:
		t.value = parseStoredTime(value)
		return nil
	case []byte:
		t.value = parseStoredTime(string(value))
		return nil
	default:
		t.value = parseStoredTime(fmt.Sprint(value))
		return nil
	}
}

func parseStoredTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed
		}
	}
	return nil
}
