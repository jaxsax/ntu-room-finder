package main

import (
	"fmt"
	"github.com/jaxsax/ntu-room-finder/pkg/crawler"
	"os"
)

func main() {
	parser := crawler.NewParser()

	file, err := os.Open("testdata/main")
	defer file.Close()

	acadSem, err := parser.FindLatestAcadSem(file)
	if err != nil {
		fmt.Printf("cant find latest acad sem: %s\n", err)
		return
	}

	fmt.Printf("k: %s val: %s\n", acadSem.Text, acadSem.Key)
}
