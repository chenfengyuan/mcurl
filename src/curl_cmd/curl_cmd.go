package curl_cmd

import (
	"net/http"
	"regexp"
	"strings"
)

// import "fmt"

func ParseCmdStr(cmd string) []string {
	re := regexp.MustCompile("([^' ][^ ]+)|(?:'((?:[^']*(?:'\"'\"')*)+)')")
	quote_re := regexp.MustCompile(`'"'"'`)
	tmp := re.FindAllStringSubmatch(cmd, -1)
	rv := make([]string, 0)
	for _, v := range tmp {
		v = v[1:]
		for _, m := range v {
			if m != "" {
				m = quote_re.ReplaceAllString(m, "'")
				rv = append(rv, m)
				break
			}
		}
		// fmt.Printf("%#v\n", v)
	}
	return rv
}

func GetHeadersFromCurlCmd(cmd string) http.Header {
	args := ParseCmdStr(cmd)
	header := http.Header{}

	for i := 0; i < len(args)-1; {
		if args[i] == "-H" {
			pair := strings.SplitN(args[i+1], ":", 2)
			key, value := pair[0], strings.TrimLeft(pair[1], " ")
			header.Add(key, value)
		}
		i += 1
	}
	return header
}
