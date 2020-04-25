package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	txtpath, err := filepath.Abs("./server/template.static.go")
	if err != nil {
		panic(err)
	}
	f, err := os.Create(txtpath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	templatePath, err := filepath.Abs("./server/index.html")
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadFile(templatePath)
	if err != nil {
		panic(err)
	}

	f.WriteString("package server\nimport \"io/ioutil\"\nvar templateStr1 = `")
	f.Write(b)
	f.WriteString("`\nfunc init() {\nb, err := ioutil.ReadFile(\"./server/index.html\")\nif err == nil {\ntemplateStr = string(b)\n}else{templateStr=templateStr1\n}}")

}
