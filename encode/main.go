package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

func main() {
	confFile, err := os.Open("./configs/export-config.toml")
	if err != nil {
		panic(err)
	}
	defer confFile.Close()

	cf, err := io.ReadAll(confFile)
	if err != nil {
		panic(err)
	}
	fmt.Println(base64.StdEncoding.EncodeToString(cf))
}
