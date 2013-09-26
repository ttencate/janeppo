package main

import (
	"encoding/json"
	"io/ioutil"
	"fmt"
	"github.com/mrjones/oauth"
)

type Config struct {
	CnsKey, CnsSecret, Follow string
	AccessToken *oauth.AccessToken
}

func main() {
	jsonBlob, ioErr := ioutil.ReadFile("../../twitter.json")
	if ioErr != nil {
		fmt.Printf("Error opening file %s: %s\n", "../../twitter.json", ioErr)
		return
	}

	var cfg Config
	jsonErr := json.Unmarshal(jsonBlob, &cfg)
	if jsonErr != nil {
		fmt.Printf("Error parsing file %s: %s\n", "../../twitter.json", jsonErr)
		return
	}
	
	//prepare oAuth data
	c := oauth.NewConsumer(
		cfg.CnsKey,
		cfg.CnsSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "http://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})
	//open twitter connection
	requestToken, url, err := c.GetRequestTokenAndUrl("oob")
	if err != nil {
		fmt.Println("twb: An error occurred when requesting the token,", err)
		return
	}
	//authenticate, make request
	fmt.Println("twb: (1) Go to: " + url)
	fmt.Println("twb: (2) Grant access, you should get back a verification code.")
	fmt.Println("twb: (3) Enter that verification code here: ")
	verificationCode := ""
	fmt.Scanln(&verificationCode)

	accessToken, err := c.AuthorizeToken(requestToken, verificationCode)
	if err != nil {
		fmt.Println("twb: An error occurred while validating your code,", err)
		return
	}
	
	cfg.AccessToken = accessToken

	//cfg is now ready to be used for requests
	//save it to a file
	
	jsonBlob, jsonErr = json.Marshal(cfg)
	if jsonErr != nil {
		fmt.Printf("Error converting to json: %s\n", jsonErr)
		return
	}
	err = ioutil.WriteFile("../../twitter.json", jsonBlob, 0600)
	if err != nil {
		fmt.Printf("Error saving file, %s\n", err)
	}
}
