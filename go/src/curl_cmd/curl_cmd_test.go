package curl_cmd

import (
	"net/http"
	"reflect"
	"testing"
)

func TestParseCmdStr(t *testing.T) {
	sample_curl_cmd := `curl -H 'A: b""' -H ''"'"'a'"'"'' -H ''"'"'a' -H 'a'"'"''`
	results := []string{`curl`, `-H`, `A: b""`, "-H", `'a'`, "-H", "'a", "-H", "a'"}
	if !reflect.DeepEqual(ParseCmdStr(sample_curl_cmd), results) {
		t.Errorf("wrong results : %#v", ParseCmdStr(sample_curl_cmd))
	}
	real_curl_cmd := `curl 'http://d.pcs.baidu.com/file/0da81910acfa33c2d9663deb0c8c98f7?fid=1042452401-250528-87083580964336&time=1432427062&rt=sh&sign=FDTAERV-DCb740ccc5511e5e8fedcff06b081203-%2F2QCMP49NgnZYsNhJYbtggc2nkQ%3D&expires=8h&prisign=GF6x2T2CVlRV01o7Pki028kp4XDNjnf6DEaBONVnjf612EcuJMuRsnKz+HfXG/Q2MvIBe085VS8lWG7BUe+61Pgqed3TucCofWlE5y7Vq0gL9lQi8Lp8+RoXS69TsChhAroHLuuZ1uozUG7s5CSdELWik7uY51B9FxkViKUca3V+ylGvcBlarHZF6DO5WfNpfcDdG3Pb85vCrjw/y/Zq5R+LUL+yo+1BQBWUZK2IEhfLg1YEdigG7V9cswL/IrQJEGbsjGZFPCY=&chkv=1&chkbd=0&chkpc=&r=573617888' -H 'Connection: keep-alive' -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8' -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.101 Safari/537.36' -H 'Referer: http://pan.baidu.com/s/1bnGZ5ev' -H 'Accept-Encoding: gzip, deflate, sdch' -H 'Accept-Language: en-US,en;q=0.8,zh-CN;q=0.6,zh-TW;q=0.4' -H 'Cookie: BAIDUID=EDB6F503404503FAA9F740D47B4F9084:FG=1; pcsett=1432513454-f006be9590dda0b71e4d10282bf0dd44' --compressed`
	real_results := []string{`curl`, `http://d.pcs.baidu.com/file/0da81910acfa33c2d9663deb0c8c98f7?fid=1042452401-250528-87083580964336&time=1432427062&rt=sh&sign=FDTAERV-DCb740ccc5511e5e8fedcff06b081203-%2F2QCMP49NgnZYsNhJYbtggc2nkQ%3D&expires=8h&prisign=GF6x2T2CVlRV01o7Pki028kp4XDNjnf6DEaBONVnjf612EcuJMuRsnKz+HfXG/Q2MvIBe085VS8lWG7BUe+61Pgqed3TucCofWlE5y7Vq0gL9lQi8Lp8+RoXS69TsChhAroHLuuZ1uozUG7s5CSdELWik7uY51B9FxkViKUca3V+ylGvcBlarHZF6DO5WfNpfcDdG3Pb85vCrjw/y/Zq5R+LUL+yo+1BQBWUZK2IEhfLg1YEdigG7V9cswL/IrQJEGbsjGZFPCY=&chkv=1&chkbd=0&chkpc=&r=573617888`, `-H`, `Connection: keep-alive`, `-H`, `Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8`, `-H`, `User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.101 Safari/537.36`, `-H`, `Referer: http://pan.baidu.com/s/1bnGZ5ev`, `-H`, `Accept-Encoding: gzip, deflate, sdch`, `-H`, `Accept-Language: en-US,en;q=0.8,zh-CN;q=0.6,zh-TW;q=0.4`, `-H`, `Cookie: BAIDUID=EDB6F503404503FAA9F740D47B4F9084:FG=1; pcsett=1432513454-f006be9590dda0b71e4d10282bf0dd44`, `--compressed`}
	if !reflect.DeepEqual(ParseCmdStr(real_curl_cmd), real_results) {
		t.Errorf("wrong results : %#v", ParseCmdStr(real_curl_cmd))
	}
}

func TestGetHeader(t *testing.T) {
	real_curl_cmd := `curl 'http://d.pcs.baidu.com/file/0da81910acfa33c2d9663deb0c8c98f7?fid=1042452401-250528-87083580964336&time=1432427062&rt=sh&sign=FDTAERV-DCb740ccc5511e5e8fedcff06b081203-%2F2QCMP49NgnZYsNhJYbtggc2nkQ%3D&expires=8h&prisign=GF6x2T2CVlRV01o7Pki028kp4XDNjnf6DEaBONVnjf612EcuJMuRsnKz+HfXG/Q2MvIBe085VS8lWG7BUe+61Pgqed3TucCofWlE5y7Vq0gL9lQi8Lp8+RoXS69TsChhAroHLuuZ1uozUG7s5CSdELWik7uY51B9FxkViKUca3V+ylGvcBlarHZF6DO5WfNpfcDdG3Pb85vCrjw/y/Zq5R+LUL+yo+1BQBWUZK2IEhfLg1YEdigG7V9cswL/IrQJEGbsjGZFPCY=&chkv=1&chkbd=0&chkpc=&r=573617888' -H 'Connection: keep-alive' -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8' -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.101 Safari/537.36' -H 'Referer: http://pan.baidu.com/s/1bnGZ5ev' -H 'Accept-Encoding: gzip, deflate, sdch' -H 'Accept-Language: en-US,en;q=0.8,zh-CN;q=0.6,zh-TW;q=0.4' -H 'Cookie: BAIDUID=EDB6F503404503FAA9F740D47B4F9084:FG=1; pcsett=1432513454-f006be9590dda0b71e4d10282bf0dd44' --compressed`
	real_results := http.Header{"Connection": []string{"keep-alive"}, "Accept": []string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"}, "User-Agent": []string{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.101 Safari/537.36"}, "Referer": []string{"http://pan.baidu.com/s/1bnGZ5ev"}, "Accept-Encoding": []string{"gzip, deflate, sdch"}, "Accept-Language": []string{"en-US,en;q=0.8,zh-CN;q=0.6,zh-TW;q=0.4"}, "Cookie": []string{"BAIDUID=EDB6F503404503FAA9F740D47B4F9084:FG=1; pcsett=1432513454-f006be9590dda0b71e4d10282bf0dd44"}}
	if tmp := GetHeadersFromCurlCmd(real_curl_cmd); !reflect.DeepEqual(tmp, real_results) {
		t.Errorf("%#v", GetHeadersFromCurlCmd(real_curl_cmd))
	}
}
