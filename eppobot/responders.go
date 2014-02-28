package eppobot

import (
	"../twitterbot"
	"code.google.com/p/go.net/html"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
)

func sayQuote(b *QuoteBot, in *IrcMessage, query []string) {
	var failMsg, successMsg string
	var fdb []Quote

	// Find out which kind of response is desired
	if len(query) <= 1 {
		// Just !collega
		//Just send a random quote from the entire QDB
		fdb = b.Qdb
		failMsg = "Die collega herinner ik me niet."
		successMsg = "Mijn collega %s zou zeggen: \"%s\""
	} else if len(query) == 2 {
		// Collega and an argument
		//We need a random quote satisfying the search query.
		//Filter the QDB to get a smaller QDB of only matching quotes.
		filter := func(q Quote) bool {
			return CaseInsContains(q.Name, strings.TrimSpace(query[1]))
		}
		fdb = ApplyFilter(b.Qdb, filter)
		failMsg = "Die collega herinner ik me niet."
		successMsg = "Mijn collega %s zou zeggen: \"%s\""
	} else if query[1] == " " {
		// Wiezei and an argument in the second part
		filter := func(q Quote) bool {
			return CaseInsContains(q.Text, strings.TrimSpace(query[2]))
		}
		fdb = ApplyFilter(b.Qdb, filter)
		failMsg = "Ik ken niemand die zoiets onfatsoenlijks zou zeggen."
		successMsg = "Mijn collega %s zou inderdaad zeggen: \"%s\""
	} else {
		// Watzei X over Y
		//First, match string to !watzei .* over .*
		person := strings.TrimSpace(query[1])
		subject := strings.TrimSpace(query[2])
		filter := func(q Quote) bool {
			return CaseInsContains(q.Name, person) && CaseInsContains(q.Text, subject)
		}
		fdb = ApplyFilter(b.Qdb, filter)
		failMsg = "Ik ken niemand die zoiets onfatsoenlijks zou zeggen."
		successMsg = "Mijn collega %s zou inderdaad zeggen: \"%s\""
	}

	// Display error on empty result set
	if len(fdb) == 0 {
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    failMsg,
		}
		return
	}

	// Return result
	i := rand.Intn(len(fdb))
	b.Output <- &IrcMessage{
		Channel: in.Channel,
		Text:    fmt.Sprintf(successMsg, fdb[i].Name, fdb[i].Text),
	}
}

func addQuote(b *QuoteBot, in *IrcMessage, query []string) {
	//Respond to !addquote
	quote := query[1:]
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
}

//Reload the QDB
func reloadDatabase(b *QuoteBot, in *IrcMessage, query []string) {
	b.Qdb = LoadQuotes(b.Quotefile)
	b.Output <- &IrcMessage{
		Channel: in.Channel,
		Text:    fmt.Sprintf("Ik bevat nu %d wijsheden van collega's.", len(b.Qdb)),
	}
}

//Failed !collega
func respondCollege(b *QuoteBot, in *IrcMessage, query []string) {
	b.Output <- &IrcMessage{
		Channel: in.Channel,
		Text:    "Ik geef helaas geen colleges meer, ik ben met pensioen!",
	}
}
func reverseQuote(b *QuoteBot, in *IrcMessage, query []string) {
	i := rand.Intn(len(b.Qdb))
	b.Output <- &IrcMessage{
		Channel: in.Channel,
		Text:    fmt.Sprintf("Mijn collega %s zou zeggen: \"%s\"", b.Qdb[i].Text, b.Qdb[i].Name),
	}
}
func selfQuote(b *QuoteBot, in *IrcMessage, query []string) {
	sayQuote(b, in, []string{"", "ikzelf"})
}

func undoAddQuote(b *QuoteBot, in *IrcMessage, query []string) {
	//Support for removing quotes after adding them
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

func measureAttachment(b *QuoteBot, in *IrcMessage, query []string) {
	size := rand.Float32()
	if in.Sender == "piet" || in.Sender == "Eggie" {
		size += 0.3
	}
	b.Output <- &IrcMessage{
		Channel: in.Channel,
		Text:    fmt.Sprintf("%3.14f cm 8%s)", size*50, strings.Repeat("=", 1+int(size*10))),
	}
}

func measureFrustration(b *QuoteBot, in *IrcMessage, query []string) {
	size := rand.Float32()
	if in.Sender == "ijbema" {
		size += 0.3
	}
	b.Output <- &IrcMessage{
		Channel: in.Channel,
		Text:    fmt.Sprintf("%3.14f cm -_-%s", size*50, strings.Repeat(";", 1+int(size*10))),
	}
}

func train(b *QuoteBot, in *IrcMessage, query []string) {
	b.Output <- &IrcMessage{Channel: in.Channel, Text: " _||__|  |  ______   ______ "}
	b.Output <- &IrcMessage{Channel: in.Channel, Text: "(        | |      | |      |"}
	b.Output <- &IrcMessage{Channel: in.Channel, Text: "/-()---() ~ ()--() ~ ()--() "}
}

func forceDisconnect(b *QuoteBot, in *IrcMessage, query []string) {
	//Panic command
	if query[1] != b.Nickname {
		return
	}
	b.Output <- &IrcCommand{
		Command:   "QUIT",
		Arguments: ":Ik ga al",
	}
	panic("Shoo'd!")
}

func rawCommand(b *QuoteBot, in *IrcMessage, query []string) {
	//Allow for entering raw irc commands in a query
	//They would be formatted like "!raw CMD args args :args args"
	if in.Channel == in.Sender {
		b.Output <- &IrcCommand{
			Command:   query[1],
			Arguments: query[2],
		}
	}
}

func giveOps(b *QuoteBot, in *IrcMessage, query []string) {
	//Allow for requesting ops in channel
	if in.Channel != in.Sender {
		b.Output <- &IrcCommand{
			Command:   "MODE",
			Arguments: fmt.Sprintf("%s +o %s", in.Channel, in.Sender),
		}
	}
}

func twitterReset(b *QuoteBot, in *IrcMessage, query []string) {
	//Various control messages for twitterbot
	b.TwitterCtl <- twitterbot.CTL_RECONNECT
	b.Output <- &IrcMessage{
		Channel: in.Channel,
		Text:    "Walvissen weggejaagd!",
	}
}

func twitterAdd(b *QuoteBot, in *IrcMessage, query []string) {
	if len(query[1]) == 0 {
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    "Daar snap ik helemaal niets van.",
		}
	}
	go func() {
		b.TwitterCtl <- twitterbot.CTL_ADD_USER
		b.TwitterCtl <- query[1]
	}()
}

func twitterRem(b *QuoteBot, in *IrcMessage, query []string) {
	if len(query[1]) == 0 {
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    "Daar snap ik helemaal niets van.",
		}
	}
	go func() {
		b.TwitterCtl <- twitterbot.CTL_DEL_USER
		b.TwitterCtl <- query[1]
	}()
}

func twitterList(b *QuoteBot, in *IrcMessage, query []string) {
	go func() {
		b.TwitterCtl <- twitterbot.CTL_LIST_USERS
	}()
}

func twitterLink(b *QuoteBot, in *IrcMessage, query []string) {

	go func() {
		b.TwitterCtl <- twitterbot.CTL_OUTPUT_LINK
		b.TwitterCtl <- query[2]
	}()
}

func shortenLink(b *QuoteBot, in *IrcMessage, query []string) {
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

func dispatchP2k(b *QuoteBot, in *IrcMessage, query []string) {
	//P2K scanner
	//This may take a bit long, so we start a separate thread for it
	go b.ReportP2k(in.Channel)
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

func findBuilding(b *QuoteBot, in *IrcMessage, query []string) {
	//RUG building finder
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
		if CaseInsContains(r, query[1]) {
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

func genericResponse(b *QuoteBot, in *IrcMessage, query []string) {
	if query[1] != b.Nickname {
		return
	}
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
}
