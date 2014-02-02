package eppobot

import (
	"../twitterbot"
	"bufio"
	"code.google.com/p/go.net/html"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
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
			Arguments: components[3][1:],
		}
		log.Println("Invited to channel", components[3][1:])
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

	//Respond to !collega
	if strings.Index(in.Text, "!collega ") == 0 || in.Text == "!collega" {
		//Going to respond with sassy quote.
		var fdb []Quote
		if in.Text == "!collega" {
			//Just send a random quote from the entire QDB
			fdb = b.Qdb
		} else {
			//We need a random quote satisfying the search query.
			//Filter the QDB to get a smaller QDB of only matching quotes.
			query := strings.TrimSpace(in.Text[8:])
			filter := func(q Quote) bool {
				return CaseInsContains(q.Name, query)
			}
			fdb = ApplyFilter(b.Qdb, filter)
		}

		if len(fdb) == 0 {
			b.Output <- &IrcMessage{
				Channel: in.Channel,
				Text:    "Die collega herinner ik me niet.",
			}
			return
		}

		i := rand.Intn(len(fdb))
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    fmt.Sprintf("Mijn collega %s zou zeggen: \"%s\"", fdb[i].Name, fdb[i].Text),
		}
		return
	}

	//Respond to !wiezei
	if components := strings.SplitN(in.Text, " ", 2); components[0] == "!wiezei" && len(components) > 1 {
		//Going to respond with insightful quote.

		//We need a random quote satisfying the search query.
		//Filter the QDB to get a smaller QDB of only matching quotes.
		filter := func(q Quote) bool {
			return CaseInsContains(q.Text, components[1])
		}
		fdb := ApplyFilter(b.Qdb, filter)

		if len(fdb) == 0 {
			b.Output <- &IrcMessage{
				Channel: in.Channel,
				Text:    "Ik ken niemand die zoiets onfatsoenlijks zou zeggen.",
			}
			return
		}

		i := rand.Intn(len(fdb))
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    fmt.Sprintf("Mijn collega %s zou inderdaad zeggen: \"%s\"", fdb[i].Name, fdb[i].Text),
		}
		return
	}

	//Respond to !watzei _ over _
	if over := strings.Index(in.Text, " over "); strings.Index(in.Text, "!watzei ") == 0 && over > 0 {
		//Going to respond with poignant quote.
		//First, match string to !watzei .* over .*
		person := strings.TrimSpace(in.Text[8:over])
		subject := strings.TrimSpace(in.Text[over+6:])

		//We need a random quote satisfying the search query.
		//Filter the QDB to get a smaller QDB of only matching quotes.
		filter := func(q Quote) bool {
			return CaseInsContains(q.Name, person) && CaseInsContains(q.Text, subject)
		}
		fdb := ApplyFilter(b.Qdb, filter)

		if len(fdb) == 0 {
			b.Output <- &IrcMessage{
				Channel: in.Channel,
				Text:    "Ik ken niemand die zoiets onfatsoenlijks zou zeggen.",
			}
			return
		}

		i := rand.Intn(len(fdb))
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    fmt.Sprintf("Mijn collega %s zou inderdaad zeggen: \"%s\"", fdb[i].Name, fdb[i].Text),
		}
		return
	}

	//Respond to !addquote
	if strings.Index(in.Text, "!addquote ") == 0 {
		quote := strings.SplitN(in.Text[10:], ": ", 2)
		//We consider certain quotes malformed and send a short help message
		//to their creator
		if len(quote) != 2 ||
			strings.Count(quote[0], ",") == 1 ||
			strings.Count(quote[0], ",") >= 3 ||
			strings.Count(quote[1], "\"") > 0 ||
			len(strings.TrimSpace(quote[0])) == 0 ||
			len(strings.TrimSpace(quote[1])) == 0 {
			b.Output <- &IrcMessage{
				Channel: in.Channel,
				Text:    "Daar snap ik helemaal niets van.",
			}
			b.Output <- &IrcMessage{
				Channel: in.Sender,
				Text:    "!addquote Naam[, activiteit,]: Blaat",
			}
			return
		}
		quote[0] = strings.TrimSpace(quote[0])
		quote[1] = strings.TrimSpace(quote[1])

		b.Qdb = append(b.Qdb, Quote{Name: quote[0], Text: quote[1]})
		log.Printf("Adding quote to QDB.\n  %s: %s\n", quote[0], quote[1])
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    fmt.Sprintf("Als ik je goed begrijp, zou %s het volgende zeggen: \"%s\".", quote[0], quote[1]),
		}
		b.SaveQuotes()
		return
	}

	//Reload the QDB
	if strings.Index(in.Text, "!herlaad") == 0 {
		b.Qdb = LoadQuotes(b.Quotefile)
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    fmt.Sprintf("Ik bevat nu %d wijsheden van collega's.", len(b.Qdb)),
		}
		return
	}

	//Failed !collega
	if strings.Index(in.Text, "!college") == 0 {
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    "Ik geef helaas geen colleges meer, ik ben met pensioen!",
		}
		return
	}
	if strings.Index(in.Text, "!collage") == 0 {
		i := rand.Intn(len(b.Qdb))
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    fmt.Sprintf("Mijn collega %s zou zeggen: \"%s\"", b.Qdb[i].Text, b.Qdb[i].Name),
		}
		return
	}
	if strings.Index(in.Text, "!janeppo") == 0 {
		b.processChatMsg(IrcMessage{
			Channel: in.Channel,
			Sender:  in.Sender,
			Text:    "!collega ikzelf",
		})
	}

	//Support for removing quotes after adding them
	if in.Text == "!undo" {
		if b.InitLen >= len(b.Qdb) {
			b.Output <- &IrcMessage{
				Channel: in.Channel,
				Text:    "Je hebt nog helemaal niks gedaan, luiwammes.",
			}
			return
		}
		log.Printf("Deleting a quote at the request of %s.\n  %s: %s\n",
			in.Sender, b.Qdb[len(b.Qdb)-1].Name, b.Qdb[len(b.Qdb)-1].Text)
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    "Ik ken een collega die nog wel een tip voor je heeft.",
		}
		b.Output <- &IrcMessage{
			Channel: in.Sender,
			Text:    fmt.Sprintf("!addquote %s: %s", b.Qdb[len(b.Qdb)-1].Name, b.Qdb[len(b.Qdb)-1].Text),
		}
		ndb := make([]Quote, len(b.Qdb)-1, len(b.Qdb)-1)
		copy(ndb, b.Qdb)
		b.Qdb = ndb
		b.SaveQuotes()
		return
	}

	//Various measurements
	if strings.Index(in.Text, "!pikk") == 0 {
		size := rand.Float32()
		if in.Sender == "piet" || in.Sender == "Eggie" {
			size += 0.3
		}
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    fmt.Sprintf("%3.14f cm 8%s)", size*50, strings.Repeat("=", 1+int(size*10))),
		}
		return
	}
	if strings.Index(in.Text, "!ijbepikk") == 0 {
		size := rand.Float32()
		if in.Sender == "ijbema" {
			size += 0.3
		}
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    fmt.Sprintf("%3.14f cm -_-%s", size*50, strings.Repeat(";", 1+int(size*10))),
		}
		return
	}

	//GANG!!!
	if strings.Index(in.Text, "gang") == 0 {
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    "GANG!!!",
		}
		return
	}
	if strings.Index(strings.ToLower(in.Text), "lazer") >= 0 {
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    "LAZERS!",
		}
		return
	}

	//P2K scanner
	if in.Text == "!sikknel" {
		//This may take a bit long, so we start a separate thread for it
		go b.ReportP2k(in.Channel)
		return
	}

	//RUG building finder
	if strings.Index(in.Text, "!waaris ") == 0 {
		query := in.Text[8:]
		results := [...]string{
			"1111. Broerstraat 5, Academic building",
			"1113. O Kijk in t Jatstraat 41/41a, Administrative Information Provision (AIV)",
			"1114. O Kijk in t Jatstraat 39, University shop University shop",
			"1121. Oude Boteringestraat 44, Office of the University Administration building",
			"1124. Oude Boteringestraat 38, Faculty of Theology and Religious studies",
			"1126. Oude Boteringestraat 34, Faculty of Arts, HOVO",
			"1131. Oude Boteringestraat 52, Faculty of Philosophy",
			"1134. Broerstraat 9, Archeology (Arts)",
			"1211. Broerstraat 4, Library",
			"1212. Poststraat 6, Archeology (Arts)",
			"1213. O Kijk in t Jatstraat 7a, University museum",
			"1214. O Kijk in t Jatstraat 5/7, Legal theory (Law)",
			"1215. O Kijk in t Jatstraat 9, Legal Institute (Law)",
			"1311. O Kijk in t Jatstraat 26, Arts/Law/Language centre Harmoniecomplex",
			"1312. O Kijk in t Jatstraat 26, Arts/Law/Language centre Harmoniecomplex",
			"1321. O Kijk in t Jatstraat 28, Editorial office UK (university newspaper)",
			"1323. Turftorenstraat 21, Legal institute (Law)",
			"1324. Kleine Kromme Elleboog 7b, University hotel University hotel",
			"1325. Uurwerkersgang 10, student counsellors, psychological counsellors, Study support",
			"2111. Grote Rozenstraat 38, Pedagogy and Educational Sciences (GMW) Nieuwenhuis building",
			"2211. Grote kruisstraat 1/2, Psychology (GMW) Heymans building",
			"2212. Grote kruisstraat 2/1, Faculty of Behavioural and Social Sciences Munting building",
			"2221. Grote Rozenstraat 1, Sociology (GMW) Bouman building",
			"2222. Grote Rozenstraat 17, Sociology (GMW)",
			"2223. Grote Rozenstraat 15, Progamma & SWI",
			"2224. Grote Rozenstraat 3, Copyshop faculty of Behavioural and Social Sciences",
			"2231. N Kijk in t Jatstraat 70, Faculty Buro",
			"3111. Antonius Deusinglaan 2, Medical Sciences (MRI centre)",
			"3126. Bloemsingel 1, Lifelines",
			"3211. Antonius Deusinglaan 1, MWF complex (UMCG)",
			"4112. Sint Walburgstraat 22a/b/c, Student facilities + KEI",
			"4123. Bloemsingel 36/36a, Faculty of Behavioural and Social Sciences",
			"4321. Pelsterstraat 23, Faculty of Arts Pelsterpand",
			"4335. A-weg 30, Arctic Centre (Arts)",
			"4336. Munnikeholm 10, USVA cultural student centre",
			"4411. Visserstraat 47/49, Health, Safety and Environment Service/Confidential advisor",
			"4429. Oude Boteringestraat 23, Faculty of Arts",
			"4432. Oude Boteringestraat 19, Van Swinderenhuis",
			"4433. Oude Boteringestraat 13, Studium Generale",
			"5111. Nijenborgh 4, Physics, Chemistry, Industrial Engineering and Management (FWN) NCC",
			"5112. Nijenborgh 4, Physics, Chemistry, Industrial Engineering and Management (FWN) NCC",
			"5113. Nijenborgh 4, Physics, Chemistry, Industrial Engineering and Management (FWN) NCC",
			"5115. Nijenborgh 4, Physics, Chemistry, Industrial Engineering and Management (FWN) NCC",
			"5114. Nijenborgh 4, Physics, Chemistry, Industrial Engineering and Management (FWN) NCC",
			"5116. Nijenborgh 4, Physics, Chemistry, Industrial Engineering and Management (FWN) NCC",
			"5117. Nijenborgh 4, Physics, Chemistry, Industrial Engineering and Management (FWN) NCC",
			"5118. Nijenborgh 4, Physics, Chemistry, Industrial Engineering and Management (FWN) NCC",
			"5143. Zernikelaan 1, Security Porters lodge",
			"5161. Nijenborgh 9, Faculty board and general offices (FWN) Bernoulliborg",
			"5172. Nijenborgh 7, Biology, Life Sciences and Technology (FWN) Linnaeusborg",
			"5211. Blauwborgje 16, Sportcentre",
			"5231. Nadorstplein 2a, Transportation Service",
			"5236. Blauwborgje 8, University Services Department",
			"5256. Blauwborgje 8-10, University Services Department and Fundamental Informatica",
			"5263. Blauwborgje 4, Aletta Jacobs hal (examination hall)",
			"5411. Nettelbosje 2, Faculty of Economics and Business Duisenberg building",
			"5415. Landleven 1, Faculty of Spatial Sciences, CIT",
			"5416. Landleven 1, Faculty of Spatial Sciences, CIT, Teacher Education (GMW)",
			"5417. Landleven 1, Faculty of Spatial Sciences, CIT",
			"5419. Landleven 12, Astronomy/Kapteyn Institute Kapteynborg",
			"5431. Nettelbosje 1, Centre for Information Technology (CIT) Zernikeborg",
			"5711. Zernikelaan 25, KVI",
		}
		for _, r := range results {
			if CaseInsContains(r, query) {
				b.Output <- &IrcMessage{
					Channel: in.Channel,
					Text:    r,
				}
				return
			}
		}
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    "Dat gebouw stond er voor mijn pensioen nog niet, geloof ik.",
		}
	}

	//Panic command
	if strings.Index(in.Text, b.Nickname+": verdwijn") == 0 {
		b.Output <- &IrcCommand{
			Command:   "QUIT",
			Arguments: ":Ik ga al",
		}
		panic("Shoo'd!")
	}

	//Allow for entering raw irc commands in a query
	//They would be formatted like "!raw CMD args args :args args"
	if components := strings.SplitN(in.Text, " ", 3); len(components) == 3 && components[0] == "!raw" && in.Channel == in.Sender {
		b.Output <- &IrcCommand{
			Command:   components[1],
			Arguments: components[2],
		}
		return
	}

	//Allow for requesting ops in channel
	if strings.Index(in.Text, "!ops") == 0 && in.Channel != in.Sender {
		b.Output <- &IrcCommand{
			Command:   "MODE",
			Arguments: fmt.Sprintf("%s +o %s", in.Channel, in.Sender),
		}
		return
	}

	//Various control messages for twitterbot
	if strings.Index(in.Text, "!fixtwitter") == 0 {
		b.TwitterCtl <- twitterbot.CTL_RECONNECT
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    "Walvissen weggejaagd!",
		}
		return
	}

	if strings.Index(in.Text, "!follow ") == 0 {
		query := in.Text[8:]
		if len(query) == 0 {
			b.Output <- &IrcMessage{
				Channel: in.Channel,
				Text:    "Daar snap ik helemaal niets van.",
			}
		}
		go func() {
			b.TwitterCtl <- twitterbot.CTL_ADD_USER
			b.TwitterCtl <- query
		}()
		return
	}
	if strings.Index(in.Text, "!unfollow ") == 0 {
		query := in.Text[10:]
		if len(query) == 0 {
			b.Output <- &IrcMessage{
				Channel: in.Channel,
				Text:    "Daar snap ik helemaal niets van.",
			}
		}
		go func() {
			b.TwitterCtl <- twitterbot.CTL_DEL_USER
			b.TwitterCtl <- query
		}()
		return
	}
	if strings.Index(in.Text, "!following") == 0 {
		go func() {
			b.TwitterCtl <- twitterbot.CTL_LIST_USERS
		}()
		return
	}
	if strings.Index(in.Text, "!link") == 0 {
		query := ""
		if strings.Index(in.Text, "!link ") == 0 {
			query = in.Text[6:]
		}
		go func() {
			b.TwitterCtl <- twitterbot.CTL_OUTPUT_LINK
			b.TwitterCtl <- query
		}()
		return
	}

	//Link shortener
	if strings.Contains(in.Text, "http") {
		//First, check if a link needs to be shortened
		components := strings.Split(in.Text, " ")
		for _, piece := range components {
			if strings.Index(piece, "http") == 0 && len(piece) > b.UrlLength {
				//This one is quite long. Shorten it
				v := url.Values{"url": {piece}}
				resp, err := http.Get("http://nazr.in/api/shorten?" + v.Encode())
				if err == nil {
					defer resp.Body.Close()
					body, err := ioutil.ReadAll(resp.Body)
					if err == nil {
						b.Output <- &IrcMessage{
							Channel: in.Channel,
							Text:    strings.TrimSpace(string(body)),
						}
					}
				}
				return
			}
		}
	}

	//Various easter eggs - add more!
	if strings.Index(in.Text, "!butterfly") == 0 {
		if in.Channel == in.Sender {
			b.Output <- &IrcMessage{
				Channel: in.Sender,
				Text:    "Dat werkt alleen in een kanaal",
			}
			return
		}
		if rand.Float32() < 0.5 {
			go func() {
				time.Sleep(120 * time.Second)
				b.Output <- &IrcCommand{
					Command:   "KICK",
					Arguments: fmt.Sprintf("%s %s :%s", in.Channel, in.Sender, "Je vlinder heeft helaas een orkaan veroorzaakt"),
				}
			}()
			log.Printf("Going to kick %s from %s in two minutes\n", in.Sender, in.Channel)
		} else {
			b.Output <- &IrcCommand{
				Command:   "NAMES",
				Arguments: in.Channel,
			}
			line, _ := b.Reader.ReadString('\n')
			if strings.Contains(line, "@"+in.Sender) {
				go func() {
					time.Sleep(120 * time.Second)
					b.Output <- &IrcCommand{
						Command:   "MODE",
						Arguments: fmt.Sprintf("%s -o %s", in.Channel, in.Sender),
					}
				}()
			} else {
				go func() {
					time.Sleep(120 * time.Second)
					b.Output <- &IrcCommand{
						Command:   "MODE",
						Arguments: fmt.Sprintf("%s +o %s", in.Channel, in.Sender),
					}
				}()
			}
			log.Printf("Going to toggle ops for %s in %s in two minutes",
				in.Sender, in.Channel)
		}
		return
	}
	if strings.Index(in.Text, "!sl") == 0 {
		b.Output <- &IrcMessage{Channel: in.Channel, Text: " _||__|  |  ______   ______ "}
		b.Output <- &IrcMessage{Channel: in.Channel, Text: "(        | |      | |      |"}
		b.Output <- &IrcMessage{Channel: in.Channel, Text: "/-()---() ~ ()--() ~ ()--() "}
		return
	}

	//Generic response
	if strings.Index(in.Text, b.Nickname+": ") == 0 {
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
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    replies[i],
		}
		return
	}
}

//This scanner connects to a P2000 site, parses it, and sends the first entry
//containing "P #" (# in 1, 2) to the channel.
func (b *QuoteBot) ReportP2k(channel string) {
	resp, err :=
		http.Get("http://www.p2000zhz-rr.nl/p2000-brandweer-groningen.html")
	if err != nil {
		log.Println("Error in HTTP-get,", err)
		return
	}
	defer resp.Body.Close()
	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Println("Error in html parser,", err)
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
			if report != "" {
				return report
			}
		}
		return ""
	}
	report := findReport(doc)
	if report == "" {
		return
	}
	report = strings.Replace(report, "\n", " ", -1)
	report = strings.Replace(report, "\r", " ", -1)
	b.Output <- &IrcMessage{
		Channel: channel,
		Text:    report,
	}
}

func IrcConnect(conf *Config) (net.Conn, *bufio.Reader) {
	conn, err := net.Dial("tcp", conf.Server)
	if err != nil {
		log.Fatalln("Error connecting to the server,", err)
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

	return conn, io
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
