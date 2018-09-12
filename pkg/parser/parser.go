package parser

import (
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"hash/fnv"
	"io"
	"strings"
)

type DefaultParser struct{}

type AcademicSemester struct {
	Key, Text string
}

type Course struct {
	Key, Text string
}

type Subject struct {
	Id    string
	Title string
	AuRaw string
	Au    float32
}

type Schedule struct {
	Index    string
	Type     string
	Group    string
	Day      string
	TimeText string
	Venue    string
	Remark   string
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

func (a *AcademicSemester) Equal(b *AcademicSemester) bool {
	return a.Key == b.Key
}

func NewParser() *DefaultParser {
	return &DefaultParser{}
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

func (p *DefaultParser) FindLatestAcadSem(body io.Reader) (*AcademicSemester, error) {
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

func (p *DefaultParser) FindCourses(body io.Reader) ([]Course, error) {
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

func (p *DefaultParser) FindSchedule(body io.Reader) ([]Schedule, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	headerMatcher := func(n *html.Node) (keep bool, exit bool) {
		isHeader := n.Type == html.ElementNode && n.DataAtom == atom.Th
		keep = isHeader

		return
	}

	lessonMatcher := func(n *html.Node) (keep bool, exit bool) {
		isTable := n.Type == html.ElementNode && n.DataAtom == atom.Table

		headerSequence := []string{"INDEX", "TYPE", "GROUP", "DAY", "TIME", "VENUE", "REMARK"}
		headers := TraverseNodes(n, headerMatcher)

		var matches bool = false
		for i, header := range headers {
			headerText := header.FirstChild.FirstChild.Data
			if headerText == headerSequence[i%len(headerSequence)] {
				matches = true
			}
		}
		keep = isTable && matches
		return
	}

	trMatcher := func(n *html.Node) (keep bool, exit bool) {
		isTr := n.Type == html.ElementNode && n.DataAtom == atom.Tr

		keep = isTr
		return
	}

	schedules := make([]Schedule, 0)
	lessonTable := TraverseNodes(doc, lessonMatcher)
	for _, subject := range lessonTable {
		rows := TraverseNodes(subject, trMatcher)
		dataRows := rows[1:]

		var cachedIndex string
		for _, row := range dataRows {
			var schedule Schedule
			for i, td := 0, row.FirstChild.NextSibling; td != nil; i, td = i+1, td.NextSibling.NextSibling {
				node := td.FirstChild.FirstChild
				var text string
				if node != nil {
					text = node.Data
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
	}
	return schedules, nil
}

func FindAttribute(attrs []html.Attribute, attr string) (string, error) {
	for _, a := range attrs {
		if a.Key == attr {
			return a.Val, nil
		}
	}
	return "", ErrCantFindAttribute
}
