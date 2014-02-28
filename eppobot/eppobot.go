package eppobot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"
)

type Config struct {
	Nickname  string
	Server    string
	Channel   string
	Quotefile string
	UrlLength int
	AutoOps   bool
	Verbose   bool
	Colors    bool
}

type Quote struct {
	Name, Text string
}

type QuoteBot struct {
	Config
	Qdb        []Quote
	Reader     *bufio.Reader
	Output     chan IrcOperation
	InitLen    int
	TwitterCtl chan string
}

type IrcMessage struct {
	Channel string
	Text    string
	Sender  string
}

type IrcCommand struct {
	Command   string
	Arguments string
}

type IrcOperation interface {
	String() string
	Type() string
}

func (m *IrcMessage) String() string {
	return fmt.Sprintf("PRIVMSG %s :%s\n", m.Channel, m.Text)
}
func (m *IrcMessage) Type() string {
	return "PRIVMSG"
}

func (o *IrcCommand) String() string {
	return o.Command + " " + o.Arguments + "\n"
}
func (o *IrcCommand) Type() string {
	return o.Command
}

func CreateBot(conf Config, reader *bufio.Reader, output chan IrcOperation, qdb []Quote) *QuoteBot {
	return &QuoteBot{
		Config:     conf,
		Qdb:        qdb,
		Reader:     reader,
		Output:     output,
		InitLen:    len(qdb),
		TwitterCtl: nil,
	}
}

func (b *QuoteBot) ChatContinuous() {
	for {
		b.ChatLine()
	}
}

func (b *QuoteBot) ChatLine() {
	//Read a line, respond if needed
	line, err := b.Reader.ReadString('\n')
	if err != nil {
		log.Panicln("Error reading from the network,", err)
	}
	if b.Verbose {
		log.Printf("%s\n", line)
	}

	components := strings.SplitN(line, " ", 4)

	//Test if this is a ping message
	if components[0] == "PING" && len(components) >= 2 {
		b.Output <- &IrcCommand{
			Command:   "PONG",
			Arguments: strings.TrimSpace(components[1]),
		}
		if b.Verbose {
			log.Print("Replying to a ping message from ", strings.TrimSpace(components[1]))
		}
	}

	if components[1] == "PRIVMSG" && len(components) >= 4 {
		b.processChatMsg(IrcMessage{
			Channel: components[2],
			Text:    strings.TrimSpace(components[3][1:]),
			Sender:  components[0][1:strings.Index(components[0], "!")],
		})
	}

	if components[1] == "INVITE" && len(components) >= 4 {
		b.Output <- &IrcCommand{
			Command:   "JOIN",
			Arguments: strings.TrimSpace(components[3][1:]),
		}
		log.Println("Invited to channel", strings.TrimSpace(components[3][1:]))
	}

	if components[1] == "JOIN" && len(components) >= 3 && b.Config.AutoOps {
		b.Output <- &IrcCommand{
			Command: "MODE",
			Arguments: fmt.Sprintf("%s +o %s",
				strings.TrimSpace(components[2][1:]),                //channel
				components[0][1:strings.Index(components[0], "!")]), //nick
		}
		if b.Verbose {
			log.Println("Automatic ops for", components[0][1:])
		}
	}
}

func (b *QuoteBot) processChatMsg(in IrcMessage) {
	if b.Verbose {
		log.Printf("Processing message %s(%s): >%s<\n", in.Sender, in.Channel, in.Text)
	}
	if in.Channel == b.Nickname {
		in.Channel = in.Sender
	}

	for re, handler := range messageToAction {
		if matches := re.FindStringSubmatch(in.Text); matches != nil {
			handler(b, &in, matches)
			return
		}
	}
}

func IrcConnect(conf *Config) (bool, net.Conn, *bufio.Reader) {
	conn, err := net.Dial("tcp", conf.Server)
	if err != nil {
		log.Println("Error connecting to the server,", err)
		return false, nil, nil
	} else {
		log.Println("Connected to server.")
	}

	fmt.Fprintf(conn, "USER gobot 8 * :Go Bot\n")
	fmt.Fprintf(conn, "NICK %s\n", conf.Nickname)

	//The server needs some time before it will accept JOIN commands
	//Hack, obviously. To be replaced by a parser for ':server 001 Welcome blah'
	time.Sleep(2000 * time.Millisecond)

	fmt.Fprintf(conn, "JOIN %s\n", conf.Channel)
	io := bufio.NewReader(conn)
	log.Println("Setup complete.")

	return true, conn, io
}

func LoadQuotes(quotefile string) []Quote {
	jsonBlob, ioErr := ioutil.ReadFile(quotefile)
	if ioErr != nil {
		log.Fatalf("Error opening file %s: %s\n", quotefile, ioErr)
	}

	var quotes []Quote
	jsonErr := json.Unmarshal(jsonBlob, &quotes)
	if jsonErr != nil {
		log.Printf("Error parsing file %s: %s\n", quotefile, jsonErr)
		log.Fatalln("Desired format: [ {\"Name\":\"...\", \"Text\":\"...\"}, " +
			"{...}, ..., {...} ]")
	}
	return quotes
}

func (b *QuoteBot) SaveQuotes() {
	jsonBlob, jsonErr := json.MarshalIndent(b.Qdb, "", "\t")
	if jsonErr != nil {
		log.Println("Error converting to JSON:", jsonErr)
		return
	}
	ioutil.WriteFile(b.Quotefile, jsonBlob, 0644)
}

func LoadConfig(file string) Config {
	jsonBlob, ioErr := ioutil.ReadFile(file)
	if ioErr != nil {
		log.Fatalf("Error opening file %s: %s\n", file, ioErr)
	}

	var conf Config
	jsonErr := json.Unmarshal(jsonBlob, &conf)
	if jsonErr != nil {
		log.Printf("Error parsing file %s: %s\n", file, jsonErr)
	}
	return conf
}

func ApplyFilter(qdb []Quote, fn func(Quote) bool) []Quote {
	var fdb []Quote
	for _, quote := range qdb {
		if fn(quote) {
			fdb = append(fdb, quote)
		}
	}
	return fdb
}

func CaseInsContains(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
