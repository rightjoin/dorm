package dorm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/rightjoin/rutl/ip"
)

func WhoStr(r *http.Request) string {

	who := WhoMap(r)

	// serialize who
	b, err := json.Marshal(who)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func WhoMap(r *http.Request) map[string]interface{} {

	who := map[string]interface{}{}

	// store general request info like ip and port
	rq := map[string]interface{}{}

	// store ip (format 122.323.23.23:92839)
	colon := strings.LastIndex(r.RemoteAddr, ":")
	if ip := r.RemoteAddr[0:colon]; strings.HasPrefix(ip, "[") && strings.HasSuffix(ip, "]") {
		rq["ip"] = ip[1 : len(ip)-1]
	} else {
		rq["ip"] = ip
	}
	rq["p"] = r.RemoteAddr[colon+1:]
	rq["raw"] = r.RemoteAddr
	rq["u"] = r.URL.String()
	rq["m"] = r.Method
	who["req"] = rq

	// store all headers except cookie
	hd := map[string]interface{}{}
	for key, arr := range r.Header {
		if key != "Cookie" {
			hd[key] = strings.Join(arr, ";")
		}
	}
	who["headers"] = hd

	// store cookie values
	ck := map[string]interface{}{}
	for _, c := range r.Cookies() {
		ck[c.Name] = c.Value
	}
	who["cookies"] = ck

	return who
}

func WhoProc(script string, kv ...interface{}) string {
	host, _ := os.Hostname()

	m := map[string]interface{}{
		"script":   script,
		"hostname": host,
		"ip":       ip.GetLocal(),
	}

	for i := 0; i < len(kv); i = i + 2 {
		j := i + 1
		k := fmt.Sprintf("%v", kv[i])
		if j < len(kv) {
			m[k] = kv[j]
		}
	}

	// serialize who
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(b)
}
