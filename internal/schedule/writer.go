package schedule

import (
	"bytes"
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/parser"
)

// Generates SQL for a list of schedules
func GenerateSQL(course *parser.Course, schedules []parser.Schedule) []byte {
	template := `INSERT INTO schedule(schedule_index, schedule_type, schedule_group, day, timeText, venue, remark)
        VALUES("%s", "%s", "%s", "%s", "%s", "%s", "%s");` + "\n"

	var sqlBuilder bytes.Buffer
	fmt.Fprintf(&sqlBuilder, "-- Schedules for %s\n", course.Text)
	fmt.Fprintf(&sqlBuilder, "BEGIN TRANSACTION;\n")
	for _, schedule := range schedules {
		fmt.Fprintf(&sqlBuilder, template,
			schedule.Index, schedule.Type, schedule.Group,
			schedule.Day, schedule.TimeText, schedule.Venue, schedule.Remark)
	}
	fmt.Fprintf(&sqlBuilder, "COMMIT;\n")
	return sqlBuilder.Bytes()
}
