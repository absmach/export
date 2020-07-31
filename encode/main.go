package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	confFile, err := os.Open("./configs/export-config.toml")
	defer confFile.Close()
	if err != nil {
		panic(err)
	}

	cf, err := ioutil.ReadAll(confFile)
	if err != nil {
		panic(err)
	}
	fmt.Println(base64.StdEncoding.EncodeToString(cf))
}
