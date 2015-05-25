package http_util

import (
	"net/http"
	"testing"
)

func TestResourceStat(t *testing.T) {
	info, err := GetResourceInfo("http://7xislp.com1.z0.glb.clouddn.com/b.mp3", http.Header{})
	if err != nil {
		t.Errorf("%v", err)
	} else {
		t.Logf("%v", info)
	}
	if info.length != 42 {
		t.Errorf("wrong content length, except 42, get %v", info.length)
	}
	if info.filename != "b.mp3" {
		t.Errorf("wrong filename, except b.mp3, get %v", info.filename)
	}
}
