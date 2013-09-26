package twitterbot

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"fmt"
	"github.com/mrjones/oauth"
)

type Tweet struct {
	User struct {
		Name  string "name"
		SName string "screen_name"
	}
	Text string "text"
}

type TwitterBot struct {
	Input  *bufio.Reader
	Output chan string
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
	//prepare oAuth data
	c := oauth.NewConsumer(
		cfg.CnsKey,
		cfg.CnsSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "http://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})

	//open stream for reading
	response, err := c.Post(
		"https://stream.twitter.com/1.1/statuses/filter.json",
		map[string]string{"follow": cfg.Follow}, &(*cfg.AccessToken))
	if err != nil {
		fmt.Println("twb: An error occurred while accessing the stream,", err)
	}
	stream := bufio.NewReader(response.Body)
	fmt.Println("twb: Bot ready, listening...")

	return &TwitterBot{
		Input:  stream,
		Output: OutputChannel,
	}
}

func (b *TwitterBot) ReadContinuous() {
	for {
		//Read a tweet
		line, err := b.Input.ReadString('\n')
		if err != nil {
			fmt.Println("twb: --- Err in stream:", err, "---")
			fmt.Println("twb: Press enter to continue, C-c to back out.")
			fmt.Scanln()
		}
		if line == "\n" {
			continue
		}
		fmt.Println("debug:", line)

		//Parse tweet
		var tweet Tweet
		jErr := json.Unmarshal([]byte(line), &tweet)
		if jErr != nil {
			fmt.Println("twb: --- Err parsing stream:", jErr, "---")
		}

		//Print tweet to sdout
		if len(tweet.User.SName) > 0 && len(tweet.Text) > 0 {
			b.Output <- fmt.Sprintf("[@%s] %s", tweet.User.SName, tweet.Text)
		}
	}
}
