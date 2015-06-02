package http_info

import (
	"fmt"
	"net/http"
	"sync"
)

func Server(info *string, info_mutex *sync.Mutex) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		info_mutex.Lock()
		fmt.Fprintf(w, "%s", *info)
		info_mutex.Unlock()
	})
	http.ListenAndServe(":8181", nil)
}
