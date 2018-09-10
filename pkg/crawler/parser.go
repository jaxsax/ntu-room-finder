package crawler

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

func (a *AcademicSemester) Equal(b *AcademicSemester) bool {
	return a.Key == b.Key
}

type Course struct {
	Key, Text string
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

func (p *DefaultParser) FindLatestAcadSem(body io.Reader) (*AcademicSemester, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return &AcademicSemester{}, err
	}

	matcher := func(n *html.Node) (keep bool, exit bool) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Select {
			keep = true
			exit = true
		}
		return
	}

	selectNodes := TraverseNodes(doc, matcher)
	for _, node := range selectNodes {
		name, err := FindAttribute(node.Attr, "name")
		if err != nil {
			continue
		}

		if name == AcadSemNameKey {
			matcher := func(n *html.Node) (keep bool, exit bool) {
				isOption := n.Type == html.ElementNode && n.DataAtom == atom.Option
				_, err := FindAttribute(n.Attr, "selected")

				keep = isOption && err == nil
				return
			}

			optionNodes := TraverseNodes(doc, matcher)
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
	}
	return &AcademicSemester{}, ErrCantFindAcadSem
}

func (p *DefaultParser) FindCourses(body io.Reader) ([]Course, error) {
	courses := make([]Course, 0)

	doc, err := html.Parse(body)
	if err != nil {
		return courses, err
	}

	matcher := func(n *html.Node) (keep bool, exit bool) {
		isSelect := n.Type == html.ElementNode && n.DataAtom == atom.Select
		name, err := FindAttribute(n.Attr, "name")

		keep = isSelect && err == nil && name == CoursesNameKey
		return
	}
	optionMatcher := func(n *html.Node) (keep bool, exit bool) {
		isOption := n.Type == html.ElementNode && n.DataAtom == atom.Option

		value, err := FindAttribute(n.Attr, "value")
		if err != nil {
			keep = false
			return
		}

		isValueEmpty := len(strings.TrimSpace(value)) == 0
		keep = isOption && !isValueEmpty

		return
	}

	selectNodes := TraverseNodes(doc, matcher)
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
