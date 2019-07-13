package main

import (
	"io/ioutil"
	"log"

	"github.com/taskcluster/taskcluster/clients/client-shell/apis"
)

func main() {
	source, err := apis.GenerateServices("http://references.taskcluster.net/manifest.json", "services", "schemas")
	if err != nil {
		log.Fatalln("error: code generation failed: ", err)
	}

	if err := ioutil.WriteFile("services.go", source, 0664); err != nil {
		log.Fatalln("error: failed to save services.go: ", err)
	}
}
