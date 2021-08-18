package Utils

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"math/bits"
	"os"
	"time"
)

func MergeMap(options PrometheusLabels, defaultOptions PrometheusLabels) PrometheusLabels {
	if options == nil {
		return defaultOptions
	}
	for key, val := range defaultOptions {
		if _, present := options[key]; !present {
			options[key] = val
		}
	}
	return options
}

func HaltOnError(logger *logrus.Logger, err error, message string) {
	if err != nil {
		logger.Error(message + "\n=> " + err.Error())
		os.Exit(1)
	}
}

func WarnOnError(logger *logrus.Logger, err error, message string, callback func()) {
	if err != nil {
		logger.Warning(message + "\n" + err.Error())
		if callback != nil {
			callback()
		}
	}
}

func StartOfDay(datetime time.Time) time.Time {
	year, month, day := datetime.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, datetime.Location())
}

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func compareArrays(old []string, new []string) []string {
	tagsChanges := make([]string, 0)
	for _, value := range old {
		if !contains(new, value) && !contains(tagsChanges, value) {
			tagsChanges = append(tagsChanges, value)
		}
	}
	for _, value := range new {
		if !contains(old, value) && !contains(tagsChanges, value) {
			tagsChanges = append(tagsChanges, value)
		}
	}
	return tagsChanges
}

func HumanDuration(seconds float64) string {
	secondsInDay := uint64((time.Hour * 24).Seconds())
	secondsInHour := uint64((time.Hour).Seconds())
	secondsInMinute := uint64((time.Minute).Seconds())

	days, remainingSeconds := bits.Div64(0, uint64(seconds), secondsInDay)
	hours, remainingSeconds := bits.Div64(0, remainingSeconds, secondsInHour)
	minutes, remainingSeconds := bits.Div64(0, remainingSeconds, secondsInMinute)

	var msg string
	if days > 0 {
		msg = fmt.Sprintf("%dj %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		msg = fmt.Sprintf("%dh %dm %ds", hours, minutes, remainingSeconds)
	} else if minutes > 0 {
		msg = fmt.Sprintf("%dm %ds", minutes, remainingSeconds)
	} else {
		msg = fmt.Sprintf("%ds", remainingSeconds)
	}
	return msg
}
