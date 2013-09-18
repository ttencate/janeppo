package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"strings"
	"time"
  "net/http"
  "code.google.com/p/go.net/html"
)

const (
	Verbose   = true
	Quotefile = "collega.txt"
)

type Quote struct {
	Name, Text string
}

type QuoteBot struct {
	Nickname string
	Qdb      []Quote
	Conn     net.Conn
	Reader   *bufio.Reader
	InitLen  int
}

func main() {
	quotes := LoadQuotes()
	//eppo := CreateBot("JanErik", "#bottest", "irc.frozenfractal.com:6667", quotes)
	eppo := CreateBot("JanEppo", "#brak", "irc.frozenfractal.com:6667", quotes)
	defer eppo.Conn.Close()

	rand.Seed(time.Now().Unix())

	for {
		eppo.ChatLine()
	}
}

func CreateBot(nickname, channel, server string, qdb []Quote) *QuoteBot {
	conn, io := IrcConnect(nickname, channel, server)
	return &QuoteBot{
		Nickname: nickname,
		Qdb:      qdb,
		Conn:     conn,
		Reader:   io,
		InitLen:  len(qdb),
	}
}

func (b *QuoteBot) ChatLine() {
	//Read a line, respond if needed

	line, err := b.Reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from the network,", err)
		panic("Network error")
	}
	if Verbose {
		fmt.Printf("%s", line)
	}

	components := strings.SplitN(line, " ", 4)

	//Test if this is a ping message
	if components[0] == "PING" {
		fmt.Fprintf(b.Conn, "PONG %s\n", components[1])
		fmt.Print("Replying to a ping message from ", components[1])
	}
  
	if components[1] == "PRIVMSG" {
    if len(components) < 4 {
      //This really shouldn't happen, but it does, so let's log it at least
      fmt.Println("WARNING: the line below seems to be malformed (components)")
      fmt.Println(line)
      fmt.Fprintf(b.Conn, "PRIVMSG erik :HALP\n")
      return
    }
		b.processChatMsg(components[2], //channel or query
			components[0][1:strings.Index(components[0], "!")], //nick
			strings.TrimSpace(components[3][1:]))               //message
	}
	
	if components[1] == "INVITE" {
    if len(components) < 4 {
      //This really shouldn't happen, but it does, so let's log it at least
      fmt.Println("WARNING: the line below seems to be malformed (components)")
      fmt.Println(line)
      fmt.Fprintf(b.Conn, "PRIVMSG erik :HALP\n")
      return
    }
		fmt.Fprintf(b.Conn, "JOIN %s\n", components[3][1:])
		fmt.Print("Invited to channel ", components[3][1:])
	}
}

func (b *QuoteBot) processChatMsg(channel, sender, message string) {
	if Verbose {
		fmt.Printf("Processing message %s(%s): >%s<\n", sender, channel, message)
	}
	if channel == b.Nickname {
		channel = sender
	}

	//Respond to !collega
	if strings.Index(message, "!collega") == 0 {
		//Going to respond with sassy quote.
		var fdb []Quote
		if message == "!collega" {
			//Just send a random quote from the entire QDB
			fdb = b.Qdb
		} else {
			//We need a random quote satisfying the search query.
			//Filter the QDB to get a smaller QDB of only matching quotes.
			query := strings.TrimSpace(message[8:])
			filter := func(q Quote) bool {
				return strings.Contains(strings.ToLower(q.Name),
					strings.ToLower(query))
			}
			fdb = ApplyFilter(b.Qdb, filter)
		}

		if len(fdb) == 0 {
			fmt.Fprintf(b.Conn, "PRIVMSG %s :Die collega herinner ik me niet.\n",
				channel)
			return
		}

		i := rand.Intn(len(fdb))
		fmt.Fprintf(b.Conn,
			"PRIVMSG %s :Mijn collega %s zou zeggen: \"%s\"\n",
			channel, fdb[i].Name, fdb[i].Text)
		return
	}

	//Respond to !wiezei
	if components := strings.SplitN(message, " ", 2); components[0] == "!wiezei" && len(components) > 1 {
		//Going to respond with insightful quote.

		//We need a random quote satisfying the search query.
		//Filter the QDB to get a smaller QDB of only matching quotes.
		filter := func(q Quote) bool {
			return strings.Contains(strings.ToLower(q.Text),
				strings.ToLower(components[1]))
		}
		fdb := ApplyFilter(b.Qdb, filter)

		if len(fdb) == 0 {
			fmt.Fprintf(b.Conn,
				"PRIVMSG %s :Ik ken niemand die zoiets onfatsoenlijks zou "+
					"zeggen.\n",
				channel)
			return
		}

		i := rand.Intn(len(fdb))
		fmt.Fprintf(b.Conn,
			"PRIVMSG %s :Mijn collega %s zou inderdaad zeggen: \"%s\"\n",
			channel, fdb[i].Name, fdb[i].Text)
		return
	}

	//Respond to !addquote
	if strings.Index(message, "!addquote ") == 0 {
		quote := strings.SplitN(message[10:], ": ", 2)
		//We consider certain quotes malformed and send a short help message
		//to their creator
		if len(quote) != 2 ||
			strings.Count(quote[0], ",") == 1 ||
			strings.Count(quote[0], ",") >= 3 ||
			strings.Count(quote[1], "\"") > 0 ||
			len(strings.TrimSpace(quote[0])) == 0 ||
			len(strings.TrimSpace(quote[1])) == 0 {
			fmt.Fprintf(b.Conn,
				"PRIVMSG %s :Daar snap ik helemaal niets van.\n",
				channel)
			fmt.Fprintf(b.Conn,
				"PRIVMSG %s :!addquote Naam[, activiteit,]: Blaat\n",
				sender)
			return
		}
		quote[0] = strings.TrimSpace(quote[0])
		quote[1] = strings.TrimSpace(quote[1])

		b.Qdb = append(b.Qdb, Quote{Name: quote[0], Text: quote[1]})
		fmt.Printf("Adding quote to QDB.\n  %s: %s\n", quote[0], quote[1])
		fmt.Fprintf(b.Conn,
			"PRIVMSG %s :Als ik je goed begrijp, zou %s het volgende "+
				"zeggen: \"%s\".\n",
			channel, quote[0], quote[1])
		b.SaveQuotes()
		return
	}

	//Failed !collega
	if strings.Index(message, "!college") == 0 {
		fmt.Fprintf(b.Conn, "PRIVMSG %s :Ik geef helaas geen colleges meer, "+
			"ik ben met pensioen!\n", channel)
    return
	}
	if strings.Index(message, "!collage") == 0 {
    i := rand.Intn(len(b.Qdb))
		fmt.Fprintf(b.Conn,
			"PRIVMSG %s :Mijn collega %s zou zeggen: \"%s\"\n",
			channel, b.Qdb[i].Text, b.Qdb[i].Name)
		return
	}
  if strings.Index(message, "!janeppo") == 0 {
    b.processChatMsg(channel, sender, "!collega ikzelf")
  }

	//Support for removing quotes after adding them
	if message == "!undo" {
		if b.InitLen >= len(b.Qdb) {
			fmt.Fprintf(b.Conn,
				"PRIVMSG %s :Je hebt nog helemaal niks gedaan, luiwammes.\n",
				channel)
			return
		}
		fmt.Fprintf(b.Conn,
			"PRIVMSG %s :Ik ken een collega die nog wel een tip voor je "+
				"heeft.\n",
			channel)
		fmt.Fprintf(b.Conn,
			"PRIVMSG %s :!addquote %s: %s\n",
			sender, b.Qdb[len(b.Qdb)-1].Name, b.Qdb[len(b.Qdb)-1].Text)
		ndb := make([]Quote, len(b.Qdb)-1, len(b.Qdb)-1)
		copy(ndb, b.Qdb)
		b.Qdb = ndb
		b.SaveQuotes()
		return
	}

	//Various measurements
	if strings.Index(message, "!pikk") == 0 {
		size := rand.Float32()
		if sender == "piet" || sender == "Eggie" {
			size += 0.3
		}
		fmt.Fprintf(b.Conn,
			"PRIVMSG %s :%3.14f cm 8%s)\n",
			channel, size*50, strings.Repeat("=", 1+int(size*10)))
		return
	}
	if strings.Index(message, "!ijbepikk") == 0 {
		size := rand.Float32()
		if sender == "ijbema" {
			size += 0.3
		}
		fmt.Fprintf(b.Conn,
			"PRIVMSG %s :%3.14f cm -_-%s\n",
			channel, size*50, strings.Repeat(";", 1+int(size*10)))
		return
	}

	//GANG!!!
	if strings.Index(message, "gang") == 0 {
		fmt.Fprintf(b.Conn, "PRIVMSG %s :GANG!!!\n", channel)
		return
	}
	if strings.Index(strings.ToLower(message), "lazer") >= 0 {
		fmt.Fprintf(b.Conn, "PRIVMSG %s :LAZERS!\n", channel)
		return
	}

  //P2K scanner
	if message == "!sikknel" {
    //This may take a bit long, so we start a separate thread for it
    go b.ReportP2k(channel)
		return
	}

	//Panic command
	if strings.Index(message, b.Nickname+": verdwijn") == 0 {
		fmt.Fprintf(b.Conn, "QUIT :Ik ga al\n")
		panic("Shoo'd!")
	}
	
	//Allow for entering raw irc commands in a query
	if strings.Index(message, "!raw ") == 0 && channel == sender {
		fmt.Fprintf(b.Conn, "%s\n", message[5:])
		return
	}
	
	//Various easter eggs - add more!
	if strings.Index(message, "!butterfly") == 0 {
    if channel == sender {
      fmt.Fprintf(b.Conn, "PRIVMSG %s :Dat werkt alleen in een kanaal\n",
        sender)
      return
    }
		if rand.Float32() < 0.5 {
		  go func() {
			  time.Sleep(120 * time.Second)
			  fmt.Fprintf(b.Conn, "KICK %s %s :%s\n", channel, sender,
				  "Je vlinder heeft helaas een orkaan veroorzaakt")
      }()
      fmt.Printf("Going to kick %s from %s in two minutes\n", sender, channel)
		} else {
		  go func() {
			  time.Sleep(120 * time.Second)
        fmt.Fprintf(b.Conn, "NAMES %s\n", channel)
        line, _ := b.Reader.ReadString('\n')
        if strings.Contains(line, "@" + sender) {
				  fmt.Fprintf(b.Conn, "MODE %s -o %s\n", channel, sender)
        } else {
				  fmt.Fprintf(b.Conn, "MODE %s +o %s\n", channel, sender)
        }
      }()
      fmt.Printf("Going to toggle ops for %s in %s in two minutes\n", 
        sender, channel)
		}
		return
	}
  if strings.Index(message, "!sl") == 0 {
    fmt.Fprintf(b.Conn, "PRIVMSG %s : _||__|  |  ______   ______ \n", channel)
    fmt.Fprintf(b.Conn, "PRIVMSG %s :(        | |      | |      |\n", channel)
    fmt.Fprintf(b.Conn, "PRIVMSG %s :/-()---() ~ ()--() ~ ()--() \n", channel)
    return
  }

	//Generic response
	if strings.Index(message, b.Nickname+": ") == 0 {
		replies := [...]string{
			"Probeer het eens met euclidische meetkunde.",
			"Weet ik veel...",
			"Vraag het een ander, ik ben met pensioen",
			"Ik zal het even aan Harm vragen.",
			"Wat zei je? Ik zat even aan Ineke te denken.",
			"Daar staat wat tegenover...",
			"Leer eerst eens spellen.",
			"Denk je echt dat ik je help na alles wat je over me gezegd hebt?",
			"Dit is meer iets voor mijn collega Moddemeyer",
			"Kun je dat verklaren?",
			"Dat is niet bevredigend.",
			"Daar kun je nog geen conclusie uit trekken.",
			"Misschien dat Jan Salvador daar meer van weet.",
			"Begrijp je de vraag eigenlijk wel?",
			"Misschien moet je het eens van de andere kant bekijken.",
			"Dat kan efficienter.",
			"Daar zie ik geen Eulerpad in.",
			"Ik denk dat ik het begrijp, maar wat doet het?",
			"Daar kun je beter een graaf bij tekenen.",
			"Ik denk dat het iets met priemgetallen te maken heeft.",
		}
		i := rand.Intn(len(replies))
		fmt.Fprintf(b.Conn, "PRIVMSG %s :%s\n", channel, replies[i])
		return
	}
}

//This scanner connects to a P2000 site, parses it, and sends the first entry
//containing "P #" (# in 1, 2) to the channel.
func (b *QuoteBot) ReportP2k(channel string) {
  resp, err :=
    http.Get("http://www.p2000zhz-rr.nl/p2000-brandweer-groningen.html")
  defer resp.Body.Close()
  if err != nil {
    fmt.Println("Error in HTTP-get,", err)
    return
  }
  doc, err := html.Parse(resp.Body)
  if err != nil {
    fmt.Println("Error in html parser,", err)
    return
  }
  //Now, make a function to recurse the tree (depth-first)
  var findReport func(n *html.Node) string
  findReport = func(n *html.Node) string {
    if n.Type == html.ElementNode && n.Data == "p" &&
      len(n.Attr) > 0 && n.Attr[0].Val == "bericht" {
      report := n.FirstChild.Data
      if strings.Contains(report, "P 1") || strings.Contains(report, "P 2") {
        return report
      }
    }
    for child := n.FirstChild; child != nil; child = child.NextSibling {
      report := findReport(child)
      if report != "" { return report }
    }
    return ""
  }
  report := findReport(doc)
  if report == "" { return }
  report = strings.Replace(report, "\n", " ", -1)
  report = strings.Replace(report, "\r", " ", -1)
  fmt.Fprintf(b.Conn, "PRIVMSG %s :%s\n", channel, report)
}

func IrcConnect(nick, ircchan, server string) (net.Conn, *bufio.Reader) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		fmt.Println("Error connecting to the server,", err)
		panic("Couldn't connect")
	} else {
		fmt.Println("Connected to server.")
	}

	fmt.Fprintf(conn, "USER gobot 8 * :Go Bot\n")
	fmt.Fprintf(conn, "NICK %s\n", nick)

	//The server needs some time before it will accept JOIN commands
	//Hack, obviously. To be replaced by a parser for ':server 001 Welcome blah'
	time.Sleep(2000 * time.Millisecond)

	fmt.Fprintf(conn, "JOIN %s\n", ircchan)
	io := bufio.NewReader(conn)
	fmt.Println("Setup complete. Chat transcript follows.")

	return conn, io
}

func LoadQuotes() []Quote {
	jsonBlob, ioErr := ioutil.ReadFile(Quotefile)
	if ioErr != nil {
		fmt.Printf("Error opening file %s: %s\n", Quotefile, ioErr)
		panic("File could not be opened")
	}

	var quotes []Quote
	jsonErr := json.Unmarshal(jsonBlob, &quotes)
	if jsonErr != nil {
		fmt.Printf("Error parsing file %s: %s\n", Quotefile, jsonErr)
		fmt.Println("Desired format: [ {\"Name\":\"...\", \"Text\":\"...\"}, " +
			"{...}, ..., {...} ]")
		panic("Couldn't fetch quotes from file")
	}
	return quotes
}

func (b *QuoteBot) SaveQuotes() {
	jsonBlob, jsonErr := json.Marshal(b.Qdb)
	if jsonErr != nil {
		fmt.Println("Error converting to JSON:", jsonErr)
		return
	}
	ioutil.WriteFile(Quotefile, jsonBlob, 0644)
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
