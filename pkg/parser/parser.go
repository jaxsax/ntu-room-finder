package parser

import (
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"hash/fnv"
	"io"
	"strconv"
	"strings"
	"time"
)

type AcademicSemester struct {
	Key, Text string
}

type Course struct {
	Key, Text string
}

type Subject struct {
	Id        string
	Title     string
	AuRaw     string
	Schedules []Schedule
}

type Schedule struct {
	Index     string
	Type      string
	Group     string
	Day       string
	Venue     string
	Remark    string
	TimeText  string
	TimeStart time.Time
	TimeEnd   time.Time
}

func (s *Schedule) Id() uint64 {
	hasher := fnv.New64()

	hasher.Write([]byte(s.Index))
	hasher.Write([]byte(s.Type))
	hasher.Write([]byte(s.Group))
	hasher.Write([]byte(s.Day))
	hasher.Write([]byte(s.TimeText))
	hasher.Write([]byte(s.Venue))

	return hasher.Sum64()
}

func (c *Course) Id() uint64 {
	hasher := fnv.New64()

	hasher.Write([]byte(c.Key))

	return hasher.Sum64()
}

func (a *Subject) Equal(b *Subject) bool {
	return a.Id == b.Id
}

func (a *AcademicSemester) Equal(b *AcademicSemester) bool {
	return a.Key == b.Key
}

var (
	ErrInvalidToken          = errors.New("parser: invalid token")
	ErrCantFindAttribute     = errors.New("parser: cannot find attribute")
	ErrCantFindAcadSem       = errors.New("parser: cannot find academic semester")
	ErrCantFindScheduleTable = errors.New("parser: cannot find tables matching schedule signature")
)

const (
	AcadSemNameKey = "acadsem"
	CoursesNameKey = "r_course_yr"
)

func TraverseNodes(doc *html.Node, matcher func(*html.Node) (bool, bool)) (nodes []*html.Node) {
	var keep, exit bool
	var f func(*html.Node)
	f = func(n *html.Node) {
		keep, exit = matcher(n)
		if keep {
			nodes = append(nodes, n)
		}
		if exit {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return nodes
}

func isSelect(n *html.Node) bool {
	return n.Type == html.ElementNode && n.DataAtom == atom.Select
}

func isOption(n *html.Node) bool {
	return n.Type == html.ElementNode && n.DataAtom == atom.Option
}

func selectMatcher(nameAttr string) func(n *html.Node) (keep bool, exit bool) {
	return func(n *html.Node) (keep bool, exit bool) {
		name, err := FindAttribute(n.Attr, "name")
		isCorrectName := err == nil && name == nameAttr

		keep = isSelect(n) && isCorrectName
		return
	}
}

func FindLatestAcadSem(body io.Reader) (*AcademicSemester, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return &AcademicSemester{}, err
	}

	optionMatcher := func(n *html.Node) (keep bool, exit bool) {
		_, err := FindAttribute(n.Attr, "selected")
		keep = isOption(n) && err == nil

		return
	}

	selectNodes := TraverseNodes(doc, selectMatcher(AcadSemNameKey))
	for _, node := range selectNodes {

		optionNodes := TraverseNodes(node, optionMatcher)
		for _, node := range optionNodes {

			text := node.FirstChild.Data
			value, err := FindAttribute(node.Attr, "value")
			if err != nil {
				return &AcademicSemester{}, err
			}

			return &AcademicSemester{
				Key:  value,
				Text: text,
			}, nil
		}
	}
	return &AcademicSemester{}, ErrCantFindAcadSem
}

func FindCourses(body io.Reader) ([]Course, error) {
	courses := make([]Course, 0)

	doc, err := html.Parse(body)
	if err != nil {
		return courses, err
	}

	optionMatcher := func(n *html.Node) (keep bool, exit bool) {
		isOption := n.Type == html.ElementNode && n.DataAtom == atom.Option
		value, err := FindAttribute(n.Attr, "value")

		isValueEmpty := err == nil && len(strings.TrimSpace(value)) == 0
		keep = isOption && !isValueEmpty

		return
	}

	selectNodes := TraverseNodes(doc, selectMatcher(CoursesNameKey))
	for _, node := range selectNodes {
		optionNodes := TraverseNodes(node, optionMatcher)
		for _, node := range optionNodes {
			text := node.FirstChild.Data
			key, err := FindAttribute(node.Attr, "value")
			if err != nil {
				return courses, ErrCantFindAttribute
			}
			courses = append(courses, Course{
				Key:  key,
				Text: text,
			})
		}
	}

	return courses, nil
}

func lessonTrMatcher(n *html.Node) (keep bool, exit bool) {
	isTr := n.Type == html.ElementNode && n.DataAtom == atom.Tr
	keep = isTr
	return
}

// Accepts a html.Node containing a table
func canParseSchedule(n *html.Node) bool {
	headerMatcher := func(n *html.Node) (keep bool, exit bool) {
		keep = n.DataAtom == atom.Th
		return
	}

	headerSequence := []string{"INDEX", "TYPE", "GROUP", "DAY", "TIME", "VENUE", "REMARK"}
	matchSequence := make([]bool, len(headerSequence))
	headers := TraverseNodes(n, headerMatcher)

	for i, header := range headers {
		headerText := header.FirstChild.FirstChild.Data
		if headerText == headerSequence[i] {
			matchSequence[i] = true
		}
	}

	for _, m := range matchSequence {
		if !m {
			return false
		}
	}

	return true
}

func parseSchedule(n *html.Node) ([]Schedule, error) {
	rows := TraverseNodes(n, lessonTrMatcher)
	schedules := make([]Schedule, 0)

	dataRows := rows[1:]
	var cachedIndex string
	for _, row := range dataRows {
		var schedule Schedule
		for i, td := 0, row.FirstChild.NextSibling; td != nil; i, td = i+1, td.NextSibling.NextSibling {
			node := td.FirstChild.FirstChild
			var text string
			if node != nil {
				text = strings.TrimSpace(node.Data)
			}
			switch i {
			case 0:
				if node != nil {
					cachedIndex = text
				}
				schedule.Index = cachedIndex
				break
			case 1:
				schedule.Type = text
				break
			case 2:
				schedule.Group = text
				break
			case 3:
				schedule.Day = text
				break
			case 4:
				schedule.TimeText = text
				if len(schedule.TimeText) <= 0 {
					break
				}
				timeText := strings.Split(schedule.TimeText, "-")
				startHour, startMinute, err := splitTime(timeText[0])
				if err != nil {
					return nil, err
				}
				schedule.TimeStart = time.Date(2018, 9, 12, startHour, startMinute, 0, 0, time.UTC)

				endHour, endMinute, err := splitTime(timeText[1])
				if err != nil {
					return nil, err
				}
				schedule.TimeEnd = time.Date(2018, 9, 12, endHour, endMinute, 0, 0, time.UTC)

				break
			case 5:
				schedule.Venue = text
				break
			case 6:
				schedule.Remark = text
				break
			default:
				fmt.Printf("unhandled index: %d\n", i)
			}
		}
		schedules = append(schedules, schedule)
	}
	return schedules, nil
}

func canParseSubject(n *html.Node) bool {
	rows := TraverseNodes(n, lessonTrMatcher)
	return len(rows) == 2
}

func parseSubject(n *html.Node) (Subject, error) {
	rows := TraverseNodes(n, lessonTrMatcher)
	row := rows[0]

	tdMatcher := func(n *html.Node) (keep bool, exit bool) {
		keep = n.DataAtom == atom.Td
		return
	}

	var subject Subject
	cols := TraverseNodes(row, tdMatcher)
	for i, col := range cols {
		text := strings.TrimSpace(col.FirstChild.FirstChild.FirstChild.Data)
		switch i {
		case 0:
			subject.Id = text
			break
		case 1:
			subject.Title = text
			break
		case 2:
			subject.AuRaw = text
			break
		}
	}

	return subject, nil
}

func FindSchedule(body io.Reader) ([]Subject, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	tableMatcher := func(n *html.Node) (keep bool, exit bool) {
		keep = n.DataAtom == atom.Table
		return
	}

	subjects := make([]Subject, 0)

	tables := TraverseNodes(doc, tableMatcher)
	var subject Subject
	for _, table := range tables {
		if canParseSchedule(table) {
			schedules, err := parseSchedule(table)
			if err != nil {
				return nil, err
			}
			subject.Schedules = schedules
			subjects = append(subjects, subject)
		} else if canParseSubject(table) {
			subject, err = parseSubject(table)
			if err != nil {
				return nil, err
			}
		}
	}
	return subjects, nil
}

// Splits time from a 24 hour format into hours and minutes
// EG: 1600 -> 16 00
func splitTime(s string) (int, int, error) {
	hourPart, err := strconv.Atoi(s[:2])
	if err != nil {
		return 0, 0, err
	}
	minutePart, err := strconv.Atoi(s[2:])
	if err != nil {
		return 0, 0, err
	}
	return hourPart, minutePart, nil
}

func FindAttribute(attrs []html.Attribute, attr string) (string, error) {
	for _, a := range attrs {
		if a.Key == attr {
			return a.Val, nil
		}
	}
	return "", ErrCantFindAttribute
}
