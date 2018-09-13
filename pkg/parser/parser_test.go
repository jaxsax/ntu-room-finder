package parser_test

import (
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/parser"
	"io"
	"os"
	"strings"
	"testing"
)

func GetFileReader(name string) io.Reader {
	f, err := os.Open(name)
	if err != nil {
		fmt.Printf("cant find %s\n", name)
	}
	return f
}

func GetAcadSemFixture() io.Reader {
	f, err := os.Open("../../testdata/main")
	if err != nil {
		fmt.Println("cant find testdata/main")
	}
	return f
}

func GetSingleScheduleFixture() io.Reader {
	f, err := os.Open("../../testdata/acc-y1-single-lesson.html")
	if err != nil {
		fmt.Println("cant find testdata/acc-y1-single-lesson")
	}
	return f
}

func TestAcadSem(t *testing.T) {
	cases := []struct {
		body        io.Reader
		expected    parser.AcademicSemester
		expectedErr error
	}{
		{strings.NewReader(``), parser.AcademicSemester{}, parser.ErrCantFindAcadSem},
		{strings.NewReader(`<select name=acadsem><option selected value=a>Hello</option></select>`),
			parser.AcademicSemester{
				Key:  "a",
				Text: "Hello",
			},
			nil},
		{strings.NewReader(`
            <select name=acadsem>
                <option selected value=a>Hello</option>
                <option selected value=b>Hello1</option>
            </select>`),
			parser.AcademicSemester{
				Key:  "a",
				Text: "Hello",
			},
			nil},
		{strings.NewReader(`<select name=acadsem><option value=a>Hello</option></select>`),
			parser.AcademicSemester{},
			parser.ErrCantFindAcadSem},
		{strings.NewReader(`<select name=acadsem></select>`),
			parser.AcademicSemester{},
			parser.ErrCantFindAcadSem},
		{GetAcadSemFixture(),
			parser.AcademicSemester{
				Key:  "2018;1",
				Text: "Acad Yr 2018 Semester 1",
			}, nil},
	}

	for i, test := range cases {
		result, err := parser.FindLatestAcadSem(test.body)
		if !result.Equal(&test.expected) {
			t.Errorf("id=%d expected=%q got=%q", i, test.expected, result)
		}
		if err != test.expectedErr {
			t.Errorf("id=%d expected=%s got=%s", i, test.expectedErr, err)
		}
	}
}

func TestCourses(t *testing.T) {
	cases := []struct {
		body           io.Reader
		expectedLength int
		expectedErr    error
	}{
		{strings.NewReader(``), 0, nil},
		{strings.NewReader(`
            <select name=r_course_yr>
                <option value=1>Course 1</option>
            </select>`),
			1,
			nil},
		{strings.NewReader(`
            <select name=r_course_yr>
                <option value=1>Course 1</option>
                <option value=1>Course 1</option>
                <option value=1>Course 1</option>
            </select>`),
			3,
			nil},
		{strings.NewReader(`
            <select name=r_course_yr>
                <option value>select something</option>
                <option value=1>Course 1</option>
            </select>`),
			1,
			nil},
	}

	for i, test := range cases {
		result, err := parser.FindCourses(test.body)
		if len(result) != test.expectedLength {
			t.Errorf("id=%d expected_length=%d got=%d", i, test.expectedLength, len(result))
		}
		if err != test.expectedErr {
			t.Errorf("id=%d expected=%s got=%s", i, test.expectedErr, err)
		}
	}
}

func TestSchedule(t *testing.T) {
	cases := []struct {
		body        io.Reader
		expected    []parser.Subject
		expectedErr error
		skip        bool
	}{
		{GetSingleScheduleFixture(),
			[]parser.Subject{
				parser.Subject{
					Id:    "AB0601",
					Title: "COMMUNICATION MANAGEMENT FUNDAMENTALS",
					Schedules: []parser.Schedule{
						parser.Schedule{
							Index: "00810", Type: "LEC/STUDIO", Group: "1",
							Day: "WED", TimeText: "1830-2130", Venue: "LT1A", Remark: "Teaching Wk11",
						},
						parser.Schedule{
							Index: "00810", Type: "LEC/STUDIO", Group: "1",
							Day: "WED", TimeText: "1830-2130", Venue: "LT2A", Remark: "Teaching Wk11",
						},
						parser.Schedule{
							Index: "00810", Type: "SEM", Group: "1",
							Day: "THU", TimeText: "0830-1030", Venue: "S4-CL1", Remark: "",
						},
					},
				},
			},
			nil, false},
		{GetFileReader("../../testdata/schedule-with-subject.html"),
			[]parser.Subject{
				parser.Subject{
					Id:    "AB0601",
					Title: "COMMUNICATION MANAGEMENT FUNDAMENTALS",
					Schedules: []parser.Schedule{
						parser.Schedule{
							Index: "00731", Type: "LEC/STUDIO", Group: "1",
							Day: "TUE", TimeText: "1830-2130", Venue: "LT26", Remark: "Teaching Wk11",
						},
					},
				},
			}, nil, false},
	}

	for i, test := range cases {
		if test.skip {
			t.Logf("Skipping id=%d", i)
			continue
		}
		result, err := parser.FindSchedule(test.body)
		if len(result) != len(test.expected) {
			t.Errorf("id=%d expected_length=%d got=%d", i, len(test.expected), len(result))
		}

		for i, s := range test.expected {
			resultI := &result[i]
			equal := resultI.Equal(&s)
			if !equal {
				t.Errorf("id=%d got=%#v", i, result[i])
			}
		}
		if err != test.expectedErr {
			t.Errorf("id=%d expected=%s got=%s", i, test.expectedErr, err)
		}
	}
}
