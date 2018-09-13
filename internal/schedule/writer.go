package schedule

import (
	"bytes"
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/parser"
	"time"
)

// Generates SQL for a list of schedules
func GenerateSQL(course *parser.Course, subjects []parser.Subject) []byte {
	template := `INSERT INTO schedule(schedule_index, schedule_type, schedule_group, day, timeText, timeStart, timeEnd, venue, remark)
        VALUES("%s", "%s", "%s", "%s", "%s", "%s", "%s", "%s", "%s");` + "\n"

	var sqlBuilder bytes.Buffer
	fmt.Fprintf(&sqlBuilder, "-- Schedules for course: %s\n", course.Text)
	fmt.Fprintf(&sqlBuilder, "BEGIN TRANSACTION;\n")
	for _, subject := range subjects {
		fmt.Fprintf(&sqlBuilder, "-- Schedules for subject: %s\n", subject.Title)
		for _, schedule := range subject.Schedules {
			fmt.Fprintf(&sqlBuilder, template,
				schedule.Index, schedule.Type, schedule.Group,
				schedule.Day,
				schedule.TimeText,
				schedule.TimeStart.Format(time.RFC3339),
				schedule.TimeEnd.Format(time.RFC3339),
				schedule.Venue, schedule.Remark)
		}
	}
	fmt.Fprintf(&sqlBuilder, "COMMIT;\n")
	return sqlBuilder.Bytes()
}
