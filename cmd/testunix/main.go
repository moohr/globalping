package main

import (
	"fmt"
	"net/url"
)

func main() {
	originURLs := []string{
		"unix://var/run/some-sock.sock",
		"unix:///var/run/some-sock.sock",
		"http://localhost:8080/ping",
		"https://www.example.com/api/v1/ping",
	}

	for _, originUrl := range originURLs {
		fmt.Println("=================================")
		urlObj, err := url.Parse(originUrl)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Origin URL: %s\n", originUrl)
		fmt.Printf("Scheme: %s\n", urlObj.Scheme)
		fmt.Printf("Host: %s\n", urlObj.Host)
		fmt.Printf("Path: %s\n", urlObj.Path)
		fmt.Printf("RawPath: %s\n", urlObj.RawPath)
		fmt.Printf("Fragment: %s\n", urlObj.Fragment)
	}

}
