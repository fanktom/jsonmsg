package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"gitlab.mi.hdm-stuttgart.de/smu/jsonmsg"
	"gitlab.mi.hdm-stuttgart.de/smu/jsonmsg/golang"
)

func main() {
	file := flag.String("file", "spec.json", "spec schema file to load")
	pack := flag.String("package", "main", "name for generated package")
	gen := flag.String("generator", "go-server", "generator to use")
	flag.Parse()

	// read spec
	buf, err := ioutil.ReadFile(*file)
	if err != nil {
		panic(err)
	}

	// parse spec
	spec, err := jsonmsg.Parse(buf)
	if err != nil {
		panic(err)
	}

	// generate src
	var src []byte
	switch *gen {
	case "go-server":
		src, err = golang.ServerPackageSrc(spec, *pack)
	default:
		err = fmt.Errorf("unknown generator: %s", *gen)
	}

	if err != nil {
		panic(err)
	}

	// print src
	fmt.Println(string(src))
}
