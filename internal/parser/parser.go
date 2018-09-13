package parser

import (
	"encoding/json"
	"fmt"
	"github.com/jaxsax/ntu-room-finder/internal/downloader"
	"github.com/jaxsax/ntu-room-finder/internal/schedule"
	"github.com/jaxsax/ntu-room-finder/pkg/parser"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

type empty struct{}

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

func Parse(p string) {
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

	courseMappings, err := getCoursesToParse(fmt.Sprintf("%s/%s", p, "mapping.json"))
	if err != nil {
		log.Fatalf("failed to find course file names %v", err)
		return
	}

	sqlIn := make(chan []byte)
	sqlDone := make(chan empty)
	scheduleErr := make(chan downloader.CourseMapping)

	go sqlCombiner(sqlIn, sqlDone, outputFile)

	parseFiles(p, courseMappings, sqlIn, scheduleErr)
	for i := 0; i < len(courseMappings); i++ {
		select {
		case something := <-scheduleErr:
			log.Printf("error parsing: %v", something)
			break
		case <-sqlDone:
			log.Printf("done %d/%d", i+1, len(courseMappings))
			break
		}
	}
}

func sqlCombiner(in chan []byte, done chan empty, outputFile *os.File) {
	for {
		generatedSQL := <-in
		log.Printf("ingested an item, there are %d items", len(in))
		outputFile.Write(generatedSQL)

		done <- empty{}
	}
}

func parseFiles(folderPath string,
	courses []downloader.CourseMapping,
	sqlIn chan []byte,
	scheduleErr chan downloader.CourseMapping) {
	for _, c := range courses {
		go processCourseFile(c, sqlIn, scheduleErr, folderPath)
	}
}

func processCourseFile(c downloader.CourseMapping,
	sqlIn chan []byte,
	scheduleErr chan downloader.CourseMapping,
	folderPath string) {

	courseFile := fmt.Sprintf("%s/%s.html", folderPath, strconv.FormatUint(c.Id(), 10))
	f, err := os.Open(courseFile)
	if err != nil {
		fmt.Printf("failed to open %s", courseFile)
		scheduleErr <- c
		return
	}

	defer f.Close()

	schedulesForCourse, err := parser.FindSchedule(f)
	if err != nil {
		fmt.Printf("failed to find schedules for %s", courseFile)
		scheduleErr <- c
		return
	}

	generateSQLForParsed(c, schedulesForCourse, sqlIn)
}

func getCoursesToParse(pathToMapping string) ([]downloader.CourseMapping, error) {
	f, err := os.Open(pathToMapping)
	if err != nil {
		return nil, err
	}

	var mappings []downloader.CourseMapping
	err = json.NewDecoder(f).Decode(&mappings)
	if err != nil {
		return nil, err
	}
	return mappings, nil
}

func generateSQLForParsed(result downloader.CourseMapping,
	subjects []parser.Subject,
	sqlIn chan []byte) {

	log.Println("dispatch generate sql")
	generatedSQL := schedule.GenerateSQL(&result.Course, subjects)
	sqlIn <- generatedSQL
}

func sigInt() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT)
	<-ch
	log.Fatal("CTRL-C; exiting")
	os.Exit(0)
}
