package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	conf "github.com/olebedev/config"
)

type config struct {
	port     string
	token    string
	referers []string
}

func initConfig(executablePath string) config {
	file, err := ioutil.ReadFile(path.Join(executablePath, "config.yml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	}
	yamlString := string(file)

	cfg, err := conf.ParseYaml(yamlString)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	}
	port, err := cfg.Int("port")

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	}

	token, err := cfg.String("token")

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	}

	referer, err := cfg.List("referer")

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(0)
	}
	var ref []string
	for _, value := range referer {
		ref = append(ref, value.(string))
	}
	return config{port: strconv.Itoa(port), token: token, referers: ref}
}
