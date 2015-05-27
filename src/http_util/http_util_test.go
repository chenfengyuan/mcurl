package http_util

import (
	"log"
	"net/http"
	"os"
	"reflect"
	"testing"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func TestResourceStat(t *testing.T) {
	info, err := GetResourceInfo("http://7xislp.com1.z0.glb.clouddn.com/b.mp3", http.Header{})
	if err != nil {
		t.Errorf("%v", err)
	} else {
		t.Logf("%v", info)
	}
	if info.length != 42 {
		t.Errorf("wrong content length, expect 42, get %v", info.length)
	}
	if info.filename != "b.mp3" {
		t.Errorf("wrong filename, expect b.mp3, get %v", info.filename)
	}
}

func TestFileDownloadInfo(t *testing.T) {
	test_file := "test_file"
	var file_size int64 = 1024*1024*1024*3 + 1024*3 + 42
	n_blocks := 1024*3 + 1
	defer func() {
		os.Remove(test_file)
		os.Remove(test_file + ".info")
	}()
	Truncate(test_file, BlockSize)
	info, err := NewFileDownloadInfo(test_file, file_size)
	if err != nil {
		t.Fatal(err)
	}
	info.Sync()
	if info.Length != file_size {
		t.Fatalf("wrong file size, expect %v, get %v", file_size, info.Length)
	}
	if info.Blocks[0] != true {
		t.Fatalf("wrong Block, the first should be true")
	}
	info.Blocks[0] = false
	if !reflect.DeepEqual(info.Blocks, make([]bool, n_blocks)) {
		t.Fatalf("wrong Blocks, should be all false")
	}
	if info.Name != test_file {
		t.Fatalf("wrong filename, expect %v, get %v", test_file, info.Name)
	}
	ranges := info.UndownloadedRanges()
	if tmp_range := (DownloadRange{0, NBlocksPerRequest * BlockSize}); ranges[0] != tmp_range {
		t.Fatalf("wrong range, expect %v, get %v", tmp_range, ranges[0])
	}
	if tmp := 31; len(ranges) != tmp {
		t.Fatalf("wrong length of ranges, expect %v, get %v", tmp, len(ranges))
	}
	info.Update(BlockSize*2-1, 1)
	ranges = info.UndownloadedRanges()
	if tmp := 32; len(ranges) != tmp {
		t.Fatalf("wrong length of ranges, expect %v, get %v", tmp, len(ranges))
	}
	if tmp_range := (DownloadRange{0, BlockSize}); ranges[0] != tmp_range {
		t.Fatalf("wrong range, expect %v, get %v", tmp_range, ranges[0])
	}
	for i := 1; i < len(ranges)-1; i++ {
		if tmp_range := (DownloadRange{int64(i-1)*NBlocksPerRequest*BlockSize + BlockSize*2, NBlocksPerRequest * BlockSize}); tmp_range != ranges[i] {
			t.Fatalf("wrong range, expect %v, get #%v %v", tmp_range, i, ranges[i])
		}
	}
	if tmp_range := (DownloadRange{30*NBlocksPerRequest*BlockSize + BlockSize*2, info.Length - 30*NBlocksPerRequest*BlockSize - BlockSize*2}); tmp_range != ranges[31] {
		t.Fatalf("wrong range, expect %v, get #%v %v", tmp_range, 31, ranges[31])
	}
	info.Sync()
	info, err = NewFileDownloadInfo(test_file, file_size)
	if err != nil {
		t.Fatal(err)
	}
	if info.Blocks[1] != true {
		t.Fatalf("wrong blocks, first element is not true")
	}

	info.Update(file_size-1, 10)
	if x := info.Blocks[len(info.Blocks)-1]; x != true {
		t.Fatalf("wrong last blocks, should be true")
	}
	// t.Logf("%#v", NewFileDownloadInfo("test.mp3"))
}
