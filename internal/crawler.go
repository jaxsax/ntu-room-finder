package internal

import (
	"bytes"
	"fmt"
	"github.com/jaxsax/ntu-room-finder/internal/schedule"
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
	sem, err := parser.FindLatestAcadSem(bytes.NewReader(mainBody))
	if err != nil {
		return &parser.AcademicSemester{}, err
	}
	return sem, nil
}

func readCourses(p *parser.DefaultParser, mainBody []byte) ([]parser.Course, error) {
	courses, err := parser.FindCourses(bytes.NewReader(mainBody))
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

func getParseFolderName() string {
	year, month, day := time.Now().UTC().Date()
	return fmt.Sprintf("%4d-%02d-%02d", year, month, day)
}

func setupSqlOutput(f string) (*os.File, error) {
	var outputFile *os.File
	if _, err := os.Stat(f); os.IsExist(err) {
		outputFile, err = os.OpenFile(f, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatalf("failed to truncate %s %v", f, err)

			return nil, err
		}
		outputFile.Truncate(0)
		outputFile.Seek(0, 0)
	} else {
		outputFile, err = os.Create(f)
		if err != nil {
			log.Fatalf("failed to create %s for writing %v", f, err)
			return nil, err
		}
	}
	return outputFile, nil
}

func Parse() {
	go sigInt()

	outputFileName := "out.sql"
	outputFile, err := setupSqlOutput(outputFileName)
	if err != nil {
		log.Fatalf("failed to setup %s %v", outputFileName, err)
		return
	}
	defer outputFile.Close()

	initSQLFile := "sql/init.sql"
	initSQL, err := ioutil.ReadFile(initSQLFile)
	if err != nil {
		log.Fatalf("failed to read %s %v", initSQLFile, err)
		return
	}

	_, err = outputFile.Write(initSQL)
	if err != nil {
		log.Fatalf("failed to write initializing sql %v", err)
		return
	}

	parseFolderName := getParseFolderName()
	err = os.Mkdir(parseFolderName, 0755)
	if err != nil {
		log.Fatalf("failed to create cache folder %v", err)
		return
	}
	mainBodyFile := fmt.Sprintf("%s/%s", parseFolderName, "main.html")
	mainBody, err := getMainBody()
	if err != nil {
		log.Fatalf("failed to get main page: %v", err)
		return
	}
	go func() {
		log.Println("writing main body to parsed folder")
		err = ioutil.WriteFile(mainBodyFile, mainBody, 0755)
		if err != nil {
			log.Fatalf("failed to write main body: %v", err)
		}
	}()

	parse := parser.NewParser()
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
	sqlOut := make(chan []byte)
	sync := make(chan time.Time, 1)

	delay := 1 * time.Second
	sync <- time.Now()

	go addCourses(courses, acadSem, courseQueue, sync)
	go crawlCourse(courseQueue, coursesOut, sync, parse)
	go sqlCombiner(sqlOut, outputFile)

	processParsedCourses(coursesOut, sync, len(courses), delay, sqlOut)
}

func sqlCombiner(in chan []byte, outputFile *os.File) {
	for {
		generatedSQL := <-in
		log.Printf("ingested an item, there are %d items", len(in))
		outputFile.Write(generatedSQL)
	}
}

func generateSQLForParsed(result crawlResult, sqlOut chan []byte) {
	log.Println("dispatch generate sql")
	generatedSQL := schedule.GenerateSQL(&result.c.course, result.result)
	sqlOut <- generatedSQL
}

func processParsedCourses(coursesOut chan crawlResult, sync chan time.Time,
	length int, delay time.Duration,
	sqlOut chan []byte) {
	for i := 0; i < length; i++ {
		crawlResult := <-coursesOut

		log.Printf("%d/%d", i+1, length)

		nextCrawl := time.Now().Add(delay)
		log.Printf("next crawl: %s\n", nextCrawl.Format(time.RFC3339))
		sync <- nextCrawl

		log.Println("dispatching worker for generating sql")
		go generateSQLForParsed(crawlResult, sqlOut)
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
		case <-time.After(time.Duration(10 * time.Second)):
			log.Println("appears to be stuck, kicking the courses again")
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
