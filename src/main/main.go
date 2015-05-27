package main

import (
	"bufio"
	"curl_cmd"
	"fmt"
	"http_util"
	"log"
	"net/http"
	"os"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func test() {
	log.Printf("%#v", http_util.Get)
	resp, err := http_util.Get("http://dldir1.qq.com/qqfile/qq/QQ7.2/14810/QQ7.2.exe", http.Header{"User-Agent": []string{http_util.ChromeUserAgent}})
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("%#v", resp.Header)
}

func main() {
	test()
	return
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
