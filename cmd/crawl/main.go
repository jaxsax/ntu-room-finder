package main

import (
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/crawler"
)

func addWork(fetcher crawler.Fetcher, url string, resultChannel chan *crawler.Result) {
	fmt.Printf("Fetching %s\n", url)
	result, err := fetcher.Fetch(url)
	if err != nil {
		fmt.Println("Failed to fetch")
	}
	resultChannel <- &result
}

func main() {
	fetcher := crawler.Create()
	resultChannel := make(chan *crawler.Result)

	go addWork(fetcher, "http://google.com", resultChannel)
	go addWork(fetcher, "http://192.168.1.11:3000", resultChannel)
	for {
		result := <-resultChannel
		fmt.Printf("%s done\n", result.Url)
	}
}
