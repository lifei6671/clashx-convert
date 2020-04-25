package server

import "io/ioutil"

var templateStr = ``

func init() {
	b, err := ioutil.ReadFile("./server/index.html")
	if err == nil {
		templateStr = string(b)
	}
}
