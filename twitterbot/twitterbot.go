package twitterbot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/mrjones/oauth"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

type User struct {
	Name        string
	Screen_Name string
	Id_Str      string
}
type Tweet struct {
	User   User
	Id_Str string
	Text   string "text"
}

type TwitterBot struct {
	Conn    *io.ReadCloser
	Input   *bufio.Reader
	Output  chan string
	Control chan string
	Config  *Config
	History []*Tweet
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
	CTL_OUTPUT_LINK   = "link"
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
		History: nil,
	}
	if !b.ReadConfig() {
		return nil
	}
	b.Connect()
	go b.ListenControl()
	go func() {
		for {
			time.Sleep(24 * time.Hour)
			b.CleanHistory()
		}
	}()
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
		b.Output <- "Walvissen vallen het schip aan!"
		log.Panicln("twb: An error occurred while accessing the stream,", err)
	}

	b.Conn = &response.Body
	b.Input = bufio.NewReader(response.Body)
	log.Println("twb: Bot ready, listening...")
}

func (b *TwitterBot) ReadContinuous() {
	for {
		//Read a tweet
		line, err := b.ReadInputLine()
		if err != nil {
			log.Println("twb: Err in stream, reconnecting:", err)
			b.Connect()
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		//Parse tweet
		var tweet Tweet
		jErr := json.Unmarshal([]byte(line), &tweet)
		if jErr != nil {
			log.Println("twb: Err parsing stream:", jErr)
		}

		r := strings.NewReplacer(
			"\n", " ",
			"\r", " ")
		tweet.Text = r.Replace(tweet.Text)

		//Print tweet to output channel
		if len(tweet.User.Screen_Name) > 0 && len(tweet.Text) > 0 {
			b.History = append(b.History, &tweet)
			b.Output <- fmt.Sprintf("[@%s] %s", tweet.User.Screen_Name, tweet.Text)
		}
	}
}

//Because of a nebulous "issue 1725" in http, ReadString doesn't return an error,
//but instead panics if the connection is closed, even if from the other side.
func (b *TwitterBot) ReadInputLine() (line string, err error) {
	defer func() {
		if pan := recover(); pan != nil {
			line = ""
			err = fmt.Errorf("%v", pan)
		}
	}()
	return b.Input.ReadString('\n')
}

func (b *TwitterBot) ListenControl() {
	for {
		switch c := <-b.Control; c {
		case CTL_ADD_USER:
			if b.AddTwit(<-b.Control) {
				b.ResetConnection()
			}
		case CTL_DEL_USER:
			if b.DelTwit(<-b.Control) {
				b.ResetConnection()
			}
		case CTL_LIST_USERS:
			b.ListTwits()
		case CTL_REREAD_CONFIG:
			b.ReadConfig()
			fallthrough
		case CTL_RECONNECT:
			b.ResetConnection()
		case CTL_OUTPUT_LINK:
			b.OutputLink(<-b.Control)
		default:
			log.Printf("twb: Ignoring invalid control <%s>\n", c)
		}
	}
}
func (b *TwitterBot) CleanHistory() {
	oldlen := len(b.History)
	if oldlen < 2 {
		log.Printf("twb: No need to clean history, %d elements remain\n", oldlen)
		return
	}

	//Only save one tweet per username (the last one)
	hmap := make(map[string]*Tweet)
	for _, t := range b.History {
		hmap[(*t).User.Screen_Name] = t
	}
	//...except for the actual last one in the slice, we'll add that later
	delete(hmap, (*(b.History[oldlen-1])).User.Screen_Name)
	//(Rewriting the loop instead causes two tweets by the last user to remain)

	//Put the saved tweets in a new slice
	hnew := make([]*Tweet, 0, len(hmap)+1)
	for _, t := range hmap {
		hnew = append(hnew, t)
	}

	//Give the bot the newly rewritten History
	//If we had tweets, append the last one to guarantee that !link is
	//correct without arguments. Duplicate tweet is not a problem,
	//and will not multiply after subsequent calls to CleanHistory.
	b.History = append(hnew, b.History[oldlen-1])

	log.Printf("twb: Cleaned history, removed %d elements, %d remain\n",
		oldlen-len(b.History), len(b.History))
}

func (b *TwitterBot) OutputLink(query string) {
	if len(b.History) == 0 {
		b.Output <- "Als ik tweets heb herhaald, kun je een link opvragen naar" +
			" de laatste tweet met '!link', of een link naar de laatste tweet" +
			" van bijvoorbeeld @ineke met '!link ineke'."
		return
	}
	if query == "ineke" {
		b.Output <- "Inderdaad. Jammer dat ze niet op Twitter zit hè.."
		return
	}
	//By now, we know there are tweets in the history
	for i := range b.History {
		//We want to search the history in reverse: latest tweet gets linked
		tweet := b.History[len(b.History)-i-1]
		if strings.Contains((*tweet).User.Screen_Name, query) {
			b.Output <- tweet.Link()
			return
		}
	}
	b.Output <- "Welnu, ik word misschien wat ouder, maar van die gebruiker" +
		" heb ik nog nooit gehoord."
}
func (t *Tweet) Link() string {
	return "https://twitter.com/" + (*t).User.Screen_Name +
		"/status/" + (*t).Id_Str
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
		log.Printf("Error opening file %s: %s\n", "twitter.json", ioErr)
		return false
	}

	jsonErr := json.Unmarshal(jsonBlob, b.Config)
	if jsonErr != nil {
		log.Printf("Error parsing file %s: %s\n", "twitter.json", jsonErr)
		return false
	}
	return true
}
func (b *TwitterBot) SaveConfig() bool {
	jsonBlob, jsonErr := json.Marshal(b.Config)
	if jsonErr != nil {
		log.Println("Error converting to JSON:", jsonErr)
		return false
	}
	return ioutil.WriteFile("twitter.json", jsonBlob, 0644) == nil
}

func (b *TwitterBot) AddTwit(twit string) bool {
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
		log.Println("twb: Can't lookup users,", err)
		return nil, false
	}
	jsonBlob, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var users []User
	jErr := json.Unmarshal(jsonBlob, &users)
	if jErr != nil {
		log.Println("twb: Can't parse users,", jErr)
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
		log.Println("twb: Can't lookup user,", err)
		return User{Name: "", Screen_Name: "", Id_Str: ""}, false
	}
	jsonBlob, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()

	var user User
	jErr := json.Unmarshal(jsonBlob, &user)
	if jErr != nil {
		log.Println("twb: Can't parse user,", jErr)
		return User{Name: "", Screen_Name: "", Id_Str: ""}, false
	}
	return user, true
}
