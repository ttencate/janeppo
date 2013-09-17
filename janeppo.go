package main

import (
  "fmt"
  "net"
  "io/ioutil"
  "bufio"
  "time"
  "math/rand"
  "strings"
  "encoding/json"
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
  Qdb []Quote
  Conn net.Conn
  Reader *bufio.Reader
  InitLen int
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
    b.processChatMsg(components[2], //channel or query
                     components[0][1:strings.Index(components[0],"!")], //nick
                     strings.TrimSpace(components[3][1:])) //message
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
  if components := strings.SplitN(message, " ", 2);
    components[0] == "!wiezei" && len(components) > 1 {
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
                  "PRIVMSG %s :Ik ken niemand die zoiets onfatsoenlijks zou " +
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
                "PRIVMSG %s :Als ik je goed begrijp, zou %s het volgende " +
                "zeggen: \"%s\".\n",
                channel, quote[0], quote[1])
    b.SaveQuotes()
    return
  }
  
  //Failed !collega
  if strings.Index(message, "!college") == 0 {
    fmt.Fprintf(b.Conn, "PRIVMSG %s :Ik geef helaas geen colleges meer, "+
      "ik ben met pensioen!\n", channel)
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
                "PRIVMSG %s :Ik ken een collega die nog wel een tip voor je " +
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
                channel, size*50, strings.Repeat("=", 1 + int(size*10)))
    return
  }
  if strings.Index(message, "!ijbepikk") == 0 {
    size := rand.Float32()
    if sender == "ijbema" {
      size += 0.3
    }
    fmt.Fprintf(b.Conn,
                "PRIVMSG %s :%3.14f cm -_-%s\n",
                channel, size*50, strings.Repeat(";", 1 + int(size*10)))
    return
  }
  
  //GANG!!!
  if strings.Index(message, "gang") == 0 {
    fmt.Fprintf(b.Conn, "PRIVMSG %s :GANG!!!\n", channel)
    return
  }
  
  if message == "!sikknel" {
    fmt.Fprintf(b.Conn, "PRIVMSG %s :PRIO 1 8091 4132 4133 PLANETENLAAN 100 " +
      "GRON Uitslaande brand\n", channel)
    return
  }
  
  //Panic command
  if strings.Index(message, b.Nickname + ": verdwijn") == 0 {
    fmt.Fprintf(b.Conn, "QUIT :Ik ga al\n")
    panic("Shoo'd!")
  }
  
  //Generic response
  if strings.Index(message, b.Nickname + ": ") == 0 {
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