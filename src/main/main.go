package main

import (
	"bufio"
	"curl_cmd"
	"fmt"
	"http_util"
	"log"
	"os"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: mcurl curl-cmd-file")
	}
	fn, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(fn)
	for scanner.Scan() {
		cmd := scanner.Text()
		url := curl_cmd.ParseCmdStr(cmd)[1]
		header := curl_cmd.GetHeadersFromCurlCmd(cmd)
		fmt.Println(http_util.GetResourceInfo(url, header))
	}
}
