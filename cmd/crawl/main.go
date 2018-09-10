package main

import (
	"bytes"
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/crawler"
	"io"
	"io/ioutil"
	"os"
)

func parseAcadSem(p *crawler.DefaultParser, f io.Reader) {
	acadSem, err := p.FindLatestAcadSem(f)
	if err != nil {
		fmt.Printf("cant find latest acad sem: %s\n", err)
		return
	}

	fmt.Printf("k: %s val: %s\n", acadSem.Text, acadSem.Key)
}

func parseCourses(p *crawler.DefaultParser, f io.Reader) {
	courses, err := p.FindCourses(f)
	if err != nil {
		fmt.Printf("cant find courses: %s\n", err)
		return
	}

	for _, course := range courses {
		fmt.Printf("value:%s text=%s\n", course.Key, course.Text)
	}
}

func main() {
	parser := crawler.NewParser()

	file, err := os.Open("testdata/main")
	if err != nil {
		fmt.Printf("cant open testdata/main %s\n", err)
		return
	}

	contents, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("cant read file %s\n", err)
	}
	defer file.Close()

	parseAcadSem(parser, bytes.NewReader(contents))
	parseCourses(parser, bytes.NewReader(contents))
}
