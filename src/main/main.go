package main

import (
	"bufio"
	"http_util"
	"log"
	"os"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func test() {
	a := []int{1, 2, 3}
	log.Print(a[0])
	a = a[1:]
	log.Print(a[0])
	a = a[1:]
	log.Print(a[0])
	a = a[1:]
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: mcurl curl-cmd-file")
	}
	cmds := []string{}
	fn, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(fn)
	for scanner.Scan() {
		cmd := scanner.Text()
		if cmd == "" {
			continue
		}
		// url := curl_cmd.ParseCmdStr(cmd)[1]
		// header := curl_cmd.GetHeadersFromCurlCmd(cmd)
		// fmt.Println(http_util.GetResourceInfo(url, header))
		cmds = append(cmds, cmd)
	}
	http_util.Run(cmds, 2)
}
