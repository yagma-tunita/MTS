package timeutil

import (
	"time"
)

var ChinaLocation = time.FixedZone("CST", 8*3600)

// Now returns current time in China timezone.
func Now() time.Time {
	return time.Now().In(ChinaLocation)
}

// ParseDate parses YYYY-MM-DD in China timezone.
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, ChinaLocation)
}

// ParseDateTime parses YYYY-MM-DD HH:MM:SS in China timezone.
func ParseDateTime(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", s, ChinaLocation)
}

// FormatDate formats time to YYYY-MM-DD.
func FormatDate(t time.Time) string {
	return t.In(ChinaLocation).Format("2006-01-02")
}

// FormatDateTime formats time to YYYY-MM-DD HH:MM:SS.
func FormatDateTime(t time.Time) string {
	return t.In(ChinaLocation).Format("2006-01-02 15:04:05")
}

// StartOfDay returns the start of the day (00:00:00) in China timezone.
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.In(ChinaLocation).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, ChinaLocation)
}

// Age calculates age from birthDate (YYYY-MM-DD) to now.
func Age(birthDate time.Time) int {
	now := Now()
	years := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		years--
	}
	return years
}
