package schedule

import (
	"bytes"
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/parser"
	"time"
)

// Generates SQL for a list of schedules
func GenerateSQL(course *parser.Course, subjects []parser.Subject) []byte {
	subjectTemplate := `INSERT INTO subject(id, schedule_index, title, rawAU)
        VALUES("%s", "%s", "%s", "%s");` + "\n"

	scheduleTemplate := `INSERT INTO schedule(schedule_index, schedule_type, schedule_group, day, timeText, timeStart, timeEnd, venue, remark)
        VALUES("%s", "%s", "%s", "%s", "%s", "%s", "%s", "%s", "%s");` + "\n"

	var sqlBuilder bytes.Buffer
	fmt.Fprintf(&sqlBuilder, "\n-- Schedules for course: %s\n", course.Text)
	fmt.Fprintf(&sqlBuilder, "BEGIN TRANSACTION;\n")
	for _, subject := range subjects {
		fmt.Fprintf(&sqlBuilder, "\n-- Schedules for subject: %s\n\n", subject.Title)
		for _, schedule := range subject.Schedules {
			fmt.Fprintf(&sqlBuilder, subjectTemplate,
				subject.Id,
				schedule.Index,
				subject.Title,
				subject.AuRaw)

			fmt.Fprintf(&sqlBuilder, scheduleTemplate,
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
