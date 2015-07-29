package main

import (
	"bufio"
	"flag"
	"fmt"
	"http_util"
	"log"
	"os"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] curl-cmd-file:\n", os.Args[0])
		flag.PrintDefaults()
	}
	num_of_workers := flag.Int("workers", 2, "num of workers")
	flag.Parse()
	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(2)
	}
	cmds := []string{}
	fn, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(fn)
	for scanner.Scan() {
		cmd := scanner.Text()
		if cmd == "" {
			continue
		}
		cmds = append(cmds, cmd)
	}

	http_util.Run(cmds, *num_of_workers)
}
