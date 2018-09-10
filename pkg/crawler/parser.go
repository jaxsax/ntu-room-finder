package crawler

import (
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
)

type Parser interface {
	Parse(body io.Reader) (result ParserResult, err error)
}

type ParserResult struct {
}

type DefaultParser struct{}

type AcademicSemester struct {
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

func FindAttribute(attrs []html.Attribute, attr string) (string, error) {
	for _, a := range attrs {
		if a.Key == attr {
			return a.Val, nil
		}
	}
	return "", ErrCantFindAttribute
}

func FindHref(attrs []html.Attribute) (string, error) {
	return FindAttribute(attrs, "href")
}

func (p *DefaultParser) Parse(body io.Reader) (*ParserResult, error) {
	z := html.NewTokenizer(body)
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			return &ParserResult{}, nil
		case tt == html.StartTagToken:
			t := z.Token()

			isAnchor := t.Data == "a"
			if isAnchor {
				href, err := FindHref(t.Attr)
				if err == nil {
					fmt.Printf("(%s) link: %s\n", t.Data, href)
				}
			}
		}
	}
}