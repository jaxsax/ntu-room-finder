package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/parser"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	mainPageURL   string = "https://wish.wis.ntu.edu.sg/webexe/owa/AUS_SCHEDULE.main"
	coursePageURL string = "https://wish.wis.ntu.edu.sg/webexe/owa/AUS_SCHEDULE.main_display1"
)

type empty struct{}

func getMainBody() ([]byte, error) {
	res, err := http.Get(mainPageURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func readLatestAcademicSemester(p *parser.DefaultParser, mainBody []byte) (*parser.AcademicSemester, error) {
	sem, err := p.FindLatestAcadSem(bytes.NewReader(mainBody))
	if err != nil {
		return &parser.AcademicSemester{}, err
	}
	return sem, nil
}

func readCourses(p *parser.DefaultParser, mainBody []byte) ([]parser.Course, error) {
	courses, err := p.FindCourses(bytes.NewReader(mainBody))
	if err != nil {
		return nil, err
	}

	return courses, nil
}

type courseWithSemester struct {
	semester parser.AcademicSemester
	course   parser.Course
}

func (c *courseWithSemester) Mixed() string {
	return fmt.Sprintf("acadsem=%s&r_course_yr=%s", c.semester.Key, c.course.Key)
}

func buildCourseInformation(s parser.AcademicSemester, c parser.Course) courseWithSemester {
	return courseWithSemester{
		semester: s,
		course:   c,
	}

}

type crawlResult struct {
	c      courseWithSemester
	result []parser.Schedule
}

func crawlCourse(in chan courseWithSemester, out chan crawlResult,
	sync chan time.Time, p *parser.DefaultParser) {
	for {
		c := <-in
		log.Printf("Processing course %s\n", c.Mixed())

		form := url.Values{
			"acadsem":       {c.semester.Key},
			"r_course_yr":   {c.course.Key},
			"r_subj_code":   {"Enter Keywords or Course Code"},
			"r_search_type": {"F"},
			"boption":       {"CLoad"},
			"staff_access":  {"False"},
		}
		body := bytes.NewBufferString(form.Encode())
		req, err := http.NewRequest("POST", coursePageURL, body)
		if err != nil {
			log.Printf("failed to build request for %v (%s)\n", c, err)
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36")
		log.Printf("Sending request for %s\n", c.course.Text)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("failed to send request for %v (%s)\n", c, err)
			continue
		}
		defer res.Body.Close()

		schedules, err := p.FindSchedule(res.Body)
		if err != nil {
			log.Printf("failed to parse body for %v (%s)\n", c, err)
			continue
		}

		out <- crawlResult{c: c, result: schedules}
	}
}

func Parse() {
	go sigInt()

	parse := parser.NewParser()

	mainBody, err := getMainBody()
	if err != nil {
		log.Fatalf("failed to get main page: %v", err)
		return
	}

	acadSem, err := readLatestAcademicSemester(parse, mainBody)
	if err != nil {
		log.Fatalf("failed to get latest acad sem: %v", err)
		return
	}

	courses, err := readCourses(parse, mainBody)
	if err != nil {
		log.Fatalf("failed to get courses information: %v", err)
		return
	}

	courseQueue := make(chan courseWithSemester, len(courses))
	coursesOut := make(chan crawlResult, len(courses))
	sync := make(chan time.Time, 1)

	delay := 5 * time.Second
	sync <- time.Now()

	go addCourses(courses, acadSem, courseQueue, sync)
	go crawlCourse(courseQueue, coursesOut, sync, parse)

	processParsedCourses(coursesOut, sync, len(courses), delay)
}

func processParsedCourses(coursesOut chan crawlResult, sync chan time.Time,
	length int,
	delay time.Duration) {
	for i := 0; i < length; i++ {
		crawlResult := <-coursesOut
		fileName := fmt.Sprintf("/tmp/%s", crawlResult.c.course.Key)
		f, err := os.Create(fileName)
		if err != nil {
			log.Printf("failed to write json %s", err)
			continue
		}

		log.Printf("writing to %s\n", fileName)
		err = json.NewEncoder(f).Encode(crawlResult.result)
		if err != nil {
			log.Println(err)
		}

		nextCrawl := time.Now().Add(delay)
		log.Printf("next crawl: %s\n", nextCrawl.Format(time.RFC3339))
		sync <- nextCrawl
	}
}

func addCourses(courses []parser.Course, acadSem *parser.AcademicSemester,
	in chan courseWithSemester,
	sync chan time.Time) {
	for _, course := range courses {
		nextCrawl := <-sync
		select {
		case <-time.After(time.Until(nextCrawl)):
			in <- buildCourseInformation(*acadSem, course)
		}
	}
}

func sigInt() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT)
	<-ch
	log.Fatal("CTRL-C; exiting")
	os.Exit(0)
}
