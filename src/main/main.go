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
	f, _ := os.OpenFile("temp", os.O_RDWR|os.O_CREATE, 0666)
	f.WriteAt([]byte{'a', 'b', 'c'}, 1000)
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
		// url := curl_cmd.ParseCmdStr(cmd)[1]
		// header := curl_cmd.GetHeadersFromCurlCmd(cmd)
		// fmt.Println(http_util.GetResourceInfo(url, header))
		cmds = append(cmds, cmd)
	}
	http_util.Run(cmds)
}
