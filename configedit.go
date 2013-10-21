package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Config struct {
	Nickname  string
	Server    string
	Channel   string
	Quotefile string
	UrlLength int
	Verbose   bool
}

func main() {
	fmt.Println("This tool will help you create a config file for Janeppo.")
	confFile := GetString("File to save to, press enter for default")
	if confFile == "" {
		confFile = "config.json"
	}
	conf := Config{
		Nickname:  GetString("Nickname for the bot"),
		Server:    GetString("IRC server, url:port"),
		Channel:   GetString("IRC channel, including '#'"),
		Quotefile: GetString("Filename of quote database"),
		UrlLength: GetInt("Length of an url above which the bot will generate a short url"),
		Verbose:   GetBool("Verbose logging"),
	}
	jsonBlob, err := json.Marshal(conf)
	if err != nil {
		fmt.Println("\nCannot make config file, ", err)
	} else {
		ioutil.WriteFile(confFile, jsonBlob, 0640)
		fmt.Println("\nConfig written to " + confFile + ".")
	}
}

func GetString(prompt string) (result string) {
	fmt.Print(prompt + " :")
	fmt.Scanln(&result)
	return
}
func GetInt(prompt string) (result int) {
	fmt.Print(prompt + " #")
	fmt.Scanf("%d", &result)
	return
}
func GetBool(prompt string) (result bool) {
	fmt.Print(prompt + " (true|false):")
	fmt.Scanf("%t", &result)
	return
}
