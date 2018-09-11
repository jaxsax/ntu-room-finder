package internal

import (
	"bytes"
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/parser"
	"io"
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

func crawlCourse(in chan courseWithSemester, out chan int, sync chan time.Time) {
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

		f, err := os.Create(fmt.Sprintf("/tmp/test/%s", c.course.Key))
		if err != nil {
			log.Printf("failed to create file for %v (%s)\n", c, err)
		}

		buffer := make([]byte, 32)
		for {
			n, err := res.Body.Read(buffer)
			if err == io.EOF {
				break
			}
			f.Write(buffer[:n])
		}
		out <- 1
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
	coursesOut := make(chan int, len(courses))
	delay := 5 * time.Second
	sync := make(chan time.Time, 1)
	sync <- time.Now()

	go addCourses(courses, acadSem, courseQueue, sync)
	go crawlCourse(courseQueue, coursesOut, sync)

	for i := 0; i < len(courses); i++ {
		courseOut := <-coursesOut
		log.Printf("finished: %d\n", courseOut)

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
