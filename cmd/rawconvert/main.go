package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"photo-viewer/internal/rawconvert"
)

func main() {
	var input string
	var output string
	flag.StringVar(&input, "input", "", "input RAW file")
	flag.StringVar(&output, "output", "", "output image file")
	flag.Parse()

	if input == "" {
		log.Fatal("-input is required")
	}
	if output == "" {
		output = filepath.Join(filepath.Dir(input), filepath.Base(input)+".jpg")
	}

	if err := rawconvert.Convert(input, output); err != nil {
		log.Fatal(err)
	}

	fmt.Println(output)
}
