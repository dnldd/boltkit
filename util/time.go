package util

import (
	"math"
	"time"

	"github.com/metakeule/fmtdate"
)

const (
	TimeFormat = "YYYY-MM-DD hh:mm:ss"
	DateFormat = "DD-MMM-YYYY"
)

// GetFutureTime extends a base time to a time in the future.
func GetFutureTime(date time.Time, days time.Duration, hours time.Duration, minutes time.Duration, seconds time.Duration) time.Time {
	duration := ((time.Hour * 24) * days) + (time.Hour * hours) +
		(time.Minute * minutes) + (time.Second * seconds)
	futureTime := date.Add(duration)
	return futureTime
}

// GetPastTime regresses a base time to a time in the past.
func GetPastTime(date time.Time, days time.Duration, hours time.Duration, minutes time.Duration, seconds time.Duration) time.Time {
	duration := ((time.Hour * 24) * days) + (time.Hour * hours) +
		(time.Minute * minutes) + (time.Second * seconds)
	pastTime := date.Add(-duration)
	return pastTime
}

// DaysBetweenRangeInclusive determines the number of days inbetween two dates,
// with both days inclusive.
func DaysBetweenRangeEndInclusive(new time.Time, old time.Time) int {
	duration := new.Sub(old)
	return int(math.Floor(duration.Hours() / 24))
}

// FormatTime formats a date time using the time format.
func FormatTime(time *time.Time) string {
	if time == nil {
		return "-"
	}
	formattedDate := fmtdate.Format(TimeFormat, *time)
	return formattedDate
}
