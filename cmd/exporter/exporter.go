package exporter

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ze-kel/summarybot/cmd/db"
)

func ComposeTextFromMessages(messages []db.Message) string {
	if len(messages) == 0 {
		return ""
	}

	var sb strings.Builder

	for _, msg := range messages {
		mDates := getDates(msg.Date)

		sb.WriteString(fmt.Sprintf("[%d.%02d.%02d %s] ", mDates.year, mDates.monthNumber, mDates.day, mDates.timeFormatted))
		sb.WriteString(fmt.Sprintf("%s %s: %s", msg.FromFirstName, msg.FromLastName, msg.Message))
		sb.WriteString("\n")
	}

	return sb.String()

}

type MessageDates struct {
	day                   int
	weekday               string
	month                 string
	monthNumber           int
	year                  int
	date                  int64
	timeFormatted         string
	timeFormattedFilename string
}

func getDates(date int64) MessageDates {
	t1 := time.Unix(date, 0).In(getLocation())

	h, m, _ := t1.Clock()

	return MessageDates{
		day:                   t1.Day(),
		month:                 t1.Month().String(),
		monthNumber:           int(t1.Month()),
		year:                  t1.Year(),
		date:                  date,
		weekday:               t1.Weekday().String(),
		timeFormatted:         fmt.Sprintf("%02d:%02d", h, m),
		timeFormattedFilename: fmt.Sprintf("%02d-%02d", h, m),
	}

}

func getLocation() *time.Location {
	tz, _ := os.LookupEnv("TIMEZONE")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Printf("Error when parsing Timezone from env %s", tz)
		return time.Now().Location()
	}
	return loc
}
