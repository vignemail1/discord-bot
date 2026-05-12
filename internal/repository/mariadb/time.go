package mariadb

import (
	"time"
)

// parseDateTime parse un timestamp MySQL/MariaDB en time.Time UTC.
// Accepte les formats avec et sans fraction de secondes.
func parseDateTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}
