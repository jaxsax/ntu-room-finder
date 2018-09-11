package parser

import (
	"errors"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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

func (a *AcademicSemester) Equal(b *AcademicSemester) bool {
	return a.Key == b.Key
}

func NewParser() *DefaultParser {
	return &DefaultParser{}
}

var (
	ErrInvalidToken      = errors.New("parser: invalid token")
	ErrCantFindAttribute = errors.New("parser: cannot find attribute")
	ErrCantFindAcadSem   = errors.New("parser: cannot find academic semester")
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

func FindAttribute(attrs []html.Attribute, attr string) (string, error) {
	for _, a := range attrs {
		if a.Key == attr {
			return a.Val, nil
		}
	}
	return "", ErrCantFindAttribute
}
