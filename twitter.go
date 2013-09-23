package twitterbot

import (
	"fmt"
	"encoding/json"
	"bufio"
	"github.com/mrjones/oauth"
)

type Tweet struct {
	User struct{
		Name  string "name"
		SName string "screen_name"
	}
	Text   string "text"
}

type TwitterBot struct {
	InputStream *bufio.Reader
	Output      chan string
}

type Config struct {
	Oauth struct {
		CnsKey, CnsSecret string
	}
	Twitter struct {
		Follow string
	}
}

func main() {
	var cfg Config
	err := gcfg.ReadFileInto(&cfg, "twitter.ini")
  b := CreateBot(cfg.Oauth.CnsKey, cfg.Oauth.CnsSecret, cnf.Twitter.Follow,
		make(chan string))
	go b.ReadContinuous()
	for {fmt.Println(<-b.Output)}
}

func CreateBot(CnsKey, CnsSecret, Twits string, OutputChannel chan string) *TwitterBot {
	//prepare oAuth data
	c := oauth.NewConsumer(
		CnsKey,
		CnsSecret,
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

	//open stream for reading
	response, err := c.Post(
		"https://stream.twitter.com/1.1/statuses/filter.json",
		map[string]string{"follow": Twits}, accessToken)
	if err != nil {
		fmt.Println("twb: An error occurred while accessing the stream,", err)
	}
	stream := bufio.NewReader(response.Body)
	
	return &TwitterBot{
		InputStream: stream,
		Output:      OutputChannel,
	}
}

func (b *TwitterBot) ReadContinuous() {
	for {
		//Read a tweet
		line, err := stream.ReadString('\n')
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
			fmt.Println("twb: Press enter to continue, C-c to back out.")
			fmt.Scanln()
		}
		
		//Print tweet to sdout
		if len(tweet.Name.Sname) > 0 && len(tweet.Text) > 0 {
			b.Output <- fmt.Sprintf("[@%s] %s", tweet.User.SName, tweet.Text)
		}
	}
}

