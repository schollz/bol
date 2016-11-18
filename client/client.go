package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mholt/archiver"
)

func main() {
	archiver.Zip.Make("output.zip", []string{"test"})
	upload()
	download()
}

func upload() {
	defer timeTrack(time.Now(), "sent!")
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	file, err := os.Open("./output.zip")
	defer file.Close()
	req, err := http.NewRequest("POST", "https://edvcfs.schollz.com/post", file)
	if err != nil {
		// handle err
	}
	req.SetBasicAuth("username", "password")
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
}

func download() {
	defer timeTrack(time.Now(), "download")
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go

	req, err := http.NewRequest("GET", "https://edvcfs.schollz.com/json", nil)
	if err != nil {
		// handle err
	}
	req.SetBasicAuth("username", "password")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	outFile, err := os.Create("username" + ".zip")
	if err != nil {
		panic(err)
	}
	// handle err
	defer outFile.Close()
	_, err = io.Copy(outFile, resp.Body)
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}