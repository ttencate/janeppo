package twitterbot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/mrjones/oauth"
	"io"
	"io/ioutil"
	"strings"
)

type User struct {
	Name        string
	Screen_Name string
	Id_Str      string
}
type Tweet struct {
	User User
	Text string "text"
}

type TwitterBot struct {
	Conn    *io.ReadCloser
	Input   *bufio.Reader
	Output  chan string
	Control chan string
	Config  *Config
}

type Config struct {
	CnsKey, CnsSecret, Follow string
	AccessToken               *oauth.AccessToken
}

const (
	CTL_REREAD_CONFIG = "reread"
	CTL_RECONNECT     = "reconn"
	CTL_ADD_USER      = "add"
	CTL_DEL_USER      = "del"
	CTL_LIST_USERS    = "list"
)

func main() {
	b := CreateBot(make(chan string), make(chan string))
	go b.ReadContinuous()
	for {
		fmt.Println(<-b.Output)
	}
}

func CreateBot(OutputChannel, ControlChannel chan string) *TwitterBot {
	b := &TwitterBot{
		Conn:    nil,
		Input:   nil,
		Output:  OutputChannel,
		Control: ControlChannel,
		Config:  new(Config),
	}
	if !b.ReadConfig() {
		return nil
	}
	b.Connect()
	go b.ListenControl()
	return b
}

func (b *TwitterBot) Connect() {
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
		b.Output <- "Walvissen vallen het schip aan!"
		panic("Unhandled error in tweetbot")
	}

	b.Conn = &response.Body
	b.Input = bufio.NewReader(response.Body)
	fmt.Println("twb: Bot ready, listening...")
}

func (b *TwitterBot) ReadContinuous() {
	//Because of a nebulous "issue 1725" in http, we can't cleanly reset the connection.
	//ReadString doesn't return an error, but instead panics if the connection is closed.
	//Hence, we catch the panic, and then reconnect.
	//And as long as we're doing that, we might as well make the panic our modus operandi.
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("twb: Recovering from panic in ReadContinuous...")
			b.Connect()
			go b.ReadContinuous()
		}
	}()
	for {
		//Read a tweet
		line, err := b.Input.ReadString('\n')
		if err != nil {
			fmt.Println("twb: --- Err in stream:", err, "---")
			fmt.Println("twb: Going to reset")
			panic("Please reset the connection")
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

func (b *TwitterBot) ListenControl() {
	for {
		switch c := <-b.Control; c {
		case CTL_ADD_USER:
			if b.AddTwit(<-b.Control) { b.ResetConnection() }
		case CTL_DEL_USER:
			if b.DelTwit(<-b.Control) { b.ResetConnection() }
		case CTL_LIST_USERS:
			b.ListTwits()
		case CTL_REREAD_CONFIG:
			b.ReadConfig()
			fallthrough
		case CTL_RECONNECT:
			b.ResetConnection()
		default:
			fmt.Printf("twb: Ignoring invalid control %d\n", c)
		}
	}
}

func (b *TwitterBot) ResetConnection() {
	//To prevent two threads from running the Reconnect function at the same time,
	//this just closes the connection. The ReadContinuous function will reconnect.
	(*b.Conn).Close()
}

func (b *TwitterBot) ReadConfig() bool {
	//Re-read the configuration file used to start the bot
	jsonBlob, ioErr := ioutil.ReadFile("twitter.json")
	if ioErr != nil {
		fmt.Printf("Error opening file %s: %s\n", "twitter.json", ioErr)
		return false
	}

	jsonErr := json.Unmarshal(jsonBlob, b.Config)
	if jsonErr != nil {
		fmt.Printf("Error parsing file %s: %s\n", "twitter.json", jsonErr)
		return false
	}
	return true
}

func (b *TwitterBot) SaveConfig() bool {
	jsonBlob, jsonErr := json.Marshal(b.Config)
	if jsonErr != nil {
		fmt.Println("Error converting to JSON:", jsonErr)
		return false
	}
	return ioutil.WriteFile("twitter.json", jsonBlob, 0644) == nil
}

func (b *TwitterBot) AddTwit(twit string) bool{
	user, success := b.userFromName(twit)
	if !success {
		b.Output <- "Die persoon ken ik niet."
		return false
	}
	//Check if it's not already there
	for _, id := range strings.Split(b.Config.Follow, ",") {
		if id == user.Id_Str {
			b.Output <- "Die volg ik al."
			return false
		}
	}
	//Add the target
	b.Config.Follow += "," + user.Id_Str
	b.SaveConfig()
	return true
}

func (b *TwitterBot) DelTwit(twit string) bool {
	user, success := b.userFromName(twit)
	if !success {
		b.Output <- "Die persoon ken ik niet."
		return false
	}
	//If we're not following that person, quit
	if strings.Index(b.Config.Follow, user.Id_Str) < 0 {
		b.Output <- "Die persoon volg ik niet."
		return false
	}
	//Make new list of followers, omitting the target
	follows := []string{}
	for _, uid := range strings.Split(b.Config.Follow, ",") {
		if uid != user.Id_Str {
			follows = append(follows, uid)
		}
	}
	//Save changes
	b.Config.Follow = strings.Join(follows, ",")
	b.SaveConfig()
	return true
}

func (b *TwitterBot) ListTwits() {
	users, success := b.usersFromNumbers(b.Config.Follow)
	if !success {
		return
	}
	usernames := make([]string, len(users))
	for i := range users {
		usernames[i] = "@" + users[i].Screen_Name
	}
	b.Output <- strings.Join(usernames, ", ")
}

func (b *TwitterBot) usersFromNumbers(numbers string) ([]User, bool) {
	//Make oauth consumer
	c := oauth.NewConsumer(
		b.Config.CnsKey,
		b.Config.CnsSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "http://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})
	//Request list of users
	response, err := c.Post(
		"https://api.twitter.com/1.1/users/lookup.json",
		map[string]string{"user_id": numbers}, b.Config.AccessToken)
	if err != nil {
		fmt.Println("twb: Can't lookup users,", err)
		return nil, false
	}
	jsonBlob, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var users []User
	jErr := json.Unmarshal(jsonBlob, &users)
	if jErr != nil {
		fmt.Println("twb: Can't parse users,", jErr)
		return nil, false
	}
	return users, true
}
func (b *TwitterBot) userFromName(name string) (User, bool) {
	//Make oauth consumer
	c := oauth.NewConsumer(
		b.Config.CnsKey,
		b.Config.CnsSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "http://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})
	//Request list of users
	response, err := c.Get(
		"https://api.twitter.com/1.1/users/show.json",
		map[string]string{"screen_name": name}, b.Config.AccessToken)
	if err != nil {
		fmt.Println("twb: Can't lookup user,", err)
		return User{Name: "", Screen_Name: "", Id_Str: ""}, false
	}
	jsonBlob, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var user User
	jErr := json.Unmarshal(jsonBlob, &user)
	if jErr != nil {
		fmt.Println("twb: Can't parse user,", jErr)
		return User{Name: "", Screen_Name: "", Id_Str: ""}, false
	}
	return user, true
}
