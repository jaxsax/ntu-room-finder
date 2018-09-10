package crawler_test

import (
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/crawler"
	"io"
	"os"
	"strings"
	"testing"
)

func GetAcadSemFixture() io.Reader {
	f, err := os.Open("../../testdata/main")
	if err != nil {
		fmt.Println("cant find testdata/main")
	}
	return f
}

func TestAcadSem(t *testing.T) {
	cases := []struct {
		body        io.Reader
		expected    crawler.AcademicSemester
		expectedErr error
	}{
		{strings.NewReader(``), crawler.AcademicSemester{}, crawler.ErrCantFindAcadSem},
		{strings.NewReader(`<select name=acadsem><option selected value=a>Hello</option></select>`),
			crawler.AcademicSemester{
				Key:  "a",
				Text: "Hello",
			},
			nil},
		{strings.NewReader(`
            <select name=acadsem>
                <option selected value=a>Hello</option>
                <option selected value=b>Hello1</option>
            </select>`),
			crawler.AcademicSemester{
				Key:  "a",
				Text: "Hello",
			},
			nil},
		{strings.NewReader(`<select name=acadsem><option value=a>Hello</option></select>`),
			crawler.AcademicSemester{},
			crawler.ErrCantFindAcadSem},
		{strings.NewReader(`<select name=acadsem></select>`),
			crawler.AcademicSemester{},
			crawler.ErrCantFindAcadSem},
		{GetAcadSemFixture(),
			crawler.AcademicSemester{
				Key:  "2018;1",
				Text: "Acad Yr 2018 Semester 1",
			}, nil},
	}

	for i, test := range cases {
		parser := crawler.NewParser()
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
		parser := crawler.NewParser()
		result, err := parser.FindCourses(test.body)
		if len(result) != test.expectedLength {
			t.Errorf("id=%d expected_length=%d got=%d", i, test.expectedLength, len(result))
		}
		if err != test.expectedErr {
			t.Errorf("id=%d expected=%s got=%s", i, test.expectedErr, err)
		}
	}
}
