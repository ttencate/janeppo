package twitterbot

import (
	"bufio"
	"io"
	"encoding/json"
	"io/ioutil"
	"fmt"
	"github.com/mrjones/oauth"
	"strings"
)

type Tweet struct {
	User struct {
		Screen_Name string
	}
	Text string "text"
}

type TwitterBot struct {
	Conn   *io.ReadCloser
	Input  *bufio.Reader
	Output chan string
	Config *Config
}

type Config struct {
	CnsKey, CnsSecret, Follow string
	AccessToken *oauth.AccessToken
}

func main() {
	jsonBlob, ioErr := ioutil.ReadFile("../twitter.json")
	if ioErr != nil {
		fmt.Printf("Error opening file %s: %s\n", "../twitter.json", ioErr)
		panic("File could not be opened")
	}

	var cfg Config
	jsonErr := json.Unmarshal(jsonBlob, &cfg)
	if jsonErr != nil {
		fmt.Printf("Error parsing file %s: %s\n", "../twitter.json", jsonErr)
		panic("Couldn't fetch config from file")
	}
	
	b := CreateBot(&cfg, make(chan string))
	go b.ReadContinuous()
	for {
		fmt.Println(<-b.Output)
	}
}

func CreateBot(cfg *Config, OutputChannel chan string) *TwitterBot {
	b :=  &TwitterBot{
		Conn:   nil,
		Input:  nil,
		Output: OutputChannel,
		Config: cfg,
	}
	b.ResetConnection()
	return b
}

func (b *TwitterBot) ResetConnection() {
	//Close connection, if it exists
	if b.Conn != nil {
		(*b.Conn).Close()
	}
	
	//Create a consumer and connect to Twitter
	//prepare oAuth data
	c := oauth.NewConsumer(
		b.Config.CnsKey,
		b.Config.CnsSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "http://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})
	//open stream for reading
	response, err := c.Post(
		"https://stream.twitter.com/1.1/statuses/filter.json",
		map[string]string{"follow": b.Config.Follow}, b.Config.AccessToken)
	if err != nil {
		fmt.Println("twb: An error occurred while accessing the stream,", err)
	}
	
	b.Conn  = &response.Body
	b.Input = bufio.NewReader(response.Body)
	fmt.Println("twb: Bot ready, listening...")
}

func (b *TwitterBot) ReadContinuous() {
	for {
		//Read a tweet
		line, err := b.Input.ReadString('\n')
		if err != nil {
			fmt.Println("twb: --- Err in stream:", err, "---")
			fmt.Println("twb: Going to reset")
			b.ResetConnection()
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fmt.Printf("debug: >%s<\n", line)

		//Parse tweet
		var tweet Tweet
		jErr := json.Unmarshal([]byte(line), &tweet)
		if jErr != nil {
			fmt.Println("twb: --- Err parsing stream:", jErr, "---")
		}
		fmt.Println(tweet)
		
		r := strings.NewReplacer(
			"\n", " ",
			"\r", " ")
		tweet.Text = r.Replace(tweet.Text)
		
		//Print tweet to output channel
		if len(tweet.User.Screen_Name) > 0 && len(tweet.Text) > 0 {
			b.Output <- fmt.Sprintf("[@%s] %s", tweet.User.Screen_Name, tweet.Text)
		}
	}
}
