package main

import (
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/crawler"
	"os"
)

func addWork(fetcher crawler.Fetcher, url string, resultChannel chan *crawler.FetcherResult) {
	fmt.Printf("Fetching %s\n", url)
	result, err := fetcher.Fetch(url)
	if err != nil {
		fmt.Println("Failed to fetch")
	}
	resultChannel <- &result
}

func main() {
	parser := crawler.NewParser()

	file, err := os.Open("data/main")
	defer file.Close()

	acadSem, err := parser.FindLatestAcadSem(file)
	if err != nil {
		fmt.Printf("cant find latest acad sem: %s\n", err)
		return
	}

	fmt.Printf("k: %s val: %s\n", acadSem.Text, acadSem.Key)
}
