package downloader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/parser"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	mainPageURL   string = "https://wish.wis.ntu.edu.sg/webexe/owa/AUS_SCHEDULE.main"
	coursePageURL string = "https://wish.wis.ntu.edu.sg/webexe/owa/AUS_SCHEDULE.main_display1"
	userAgent     string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36"
)

var (
	ErrFailedToDownloadBody    = errors.New("downloader: failed to download main body")
	ErrEnsureCacheFolderExist  = errors.New("downloader: failed trying to ensure cache folder exists")
	ErrParsingAcademicSemester = errors.New("downloader: failed to parse latest academic semester")
	ErrParsingCourses          = errors.New("downloader: failed to parse courses")
	ErrDownloadingCourses      = errors.New("downloader: failed to download courses")
)

func DownloadMainBody(url string) ([]byte, error) {
	res, err := http.Get(url)
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

func store(path string, body []byte) error {
	log.Printf("storing %s", path)
	err := ioutil.WriteFile(path, body, 0755)
	if err != nil {
		return err
	}
	return nil
}

func folderCachePath() string {
	year, month, day := time.Now().UTC().Date()
	return fmt.Sprintf("%4d-%02d-%02d", year, month, day)
}

func ensureFolderExists(path string) error {
	pathStat, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.Mkdir(path, 0755)
		if err != nil {
			return err
		}
	} else if !pathStat.IsDir() {
		err = os.Mkdir(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

type courseLink struct {
	semester parser.AcademicSemester
	course   parser.Course
}

func buildCourseLink(academicSemester parser.AcademicSemester,
	courses []parser.Course) []*courseLink {
	courseLinks := make([]*courseLink, len(courses))
	for i, course := range courses {
		link := &courseLink{
			semester: academicSemester,
			course:   course,
		}
		courseLinks[i] = link
	}
	return courseLinks
}

func DownloadAndStoreCourses(courses []*courseLink, delay time.Duration,
	errChan chan courseLink,
	folderPath string) {

	for i, link := range courses {
		courseId := link.course.Id()
		log.Printf("%d/%d: %s\n", i+1, len(courses), link.course.Text)
		body, err := DownloadCourse(*link, coursePageURL)
		if err != nil {
			errChan <- *link
			continue
		}

		courseBodyPath := fmt.Sprintf("%s/%s.html",
			folderPath,
			strconv.FormatUint(courseId, 10))

		err = store(courseBodyPath, body)
		if err != nil {
			errChan <- *link
			continue
		}

		time.Sleep(delay)
	}
}

func DownloadCourse(c courseLink, coursePageURL string) ([]byte, error) {
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
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	log.Printf("Sending request for %s\n", c.course.Text)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("failed to send request for %v (%s)\n", c, err)
		return nil, err
	}
	log.Println("Request complete")
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		log.Println("Copying response into body")
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return bodyBytes, nil
	}
	return nil, fmt.Errorf("error: %s", res.Status)
}

type courseMapping struct {
	parser.Course
	Index uint64
}

func CreateCourseMapping(path string, links []*courseLink) error {
	log.Println("creating course mapping")
	mappings := make([]courseMapping, len(links))
	for i, link := range links {
		mappings[i] = courseMapping{Course: link.course, Index: link.course.Id()}
	}

	if _, err := os.Stat(path); os.IsExist(err) {
		err := os.Remove(path)
		if err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")
	err = encoder.Encode(&mappings)
	if err != nil {
		return err
	}
	return nil
}

func parseLatestAcademicSemester(mainBody *[]byte) (*parser.AcademicSemester, error) {
	sem, err := parser.FindLatestAcadSem(bytes.NewReader(*mainBody))
	if err != nil {
		return &parser.AcademicSemester{}, err
	}
	return sem, nil
}

func parseCourses(mainBody *[]byte) ([]parser.Course, error) {
	courses, err := parser.FindCourses(bytes.NewReader(*mainBody))
	if err != nil {
		return nil, err
	}

	return courses, nil
}

func Download() {
	cachedFolderPath := folderCachePath()

	err := ensureFolderExists(cachedFolderPath)
	if err != nil {
		log.Fatalf("%v: %v", ErrEnsureCacheFolderExist, err)
		return
	}

	mainBody, err := DownloadMainBody(mainPageURL)
	if err != nil {
		log.Fatalf("%v: %v", ErrFailedToDownloadBody, err)
		return
	}

	cachedMainBody := fmt.Sprintf("%s/%s", cachedFolderPath, "main.html")
	store(cachedMainBody, mainBody)

	latestAcademicSemester, err := parseLatestAcademicSemester(&mainBody)
	if err != nil {
		log.Fatalf("%v: %v", ErrParsingAcademicSemester, err)
		return
	}

	courses, err := parseCourses(&mainBody)
	if err != nil {
		log.Fatalf("%v: %v", ErrParsingCourses, err)
		return
	}
	courseLinks := buildCourseLink(*latestAcademicSemester, courses)
	delay := 1 * time.Second
	errChan := make(chan courseLink)
	jsonFileName := fmt.Sprintf("%s/%s", cachedFolderPath, "mapping.json")

	err = CreateCourseMapping(jsonFileName, courseLinks)
	if err != nil {
		log.Fatalf("failed to create mapping: %v", err)
		return
	}
	DownloadAndStoreCourses(courseLinks, delay, errChan, cachedFolderPath)

	for len(errChan) > 0 {
		course := <-errChan
		log.Printf("encountered errors on: %v", course)
	}
}
