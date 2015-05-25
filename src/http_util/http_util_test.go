package http_util

import (
	"net/http"
	"testing"
)

func TestResourceStat(t *testing.T) {
	stat, err := GetResourceStat("http://7xislp.com1.z0.glb.clouddn.com/a.mp3", http.Header{})
	if err != nil {
		t.Errorf("%v", err)
	} else {
		t.Logf("%v", stat)
	}
	if stat.length != 214483 {
		t.Errorf("wrong content length, except 42, get %v", stat.length)
	}
	if stat.filename != "a.mp3" {
		t.Errorf("wrong filename, except a.mp3, get %v", stat.filename)
	}
}
