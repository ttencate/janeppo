package main

import (
	je "./eppobot"
	"./twitterbot"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"
)

func main() {
	//Read config
	var confFile string
	flag.StringVar(&confFile, "config", "config.json", "Name of configuration file")
	flag.Parse()
	conf := je.LoadConfig(confFile)

	//Prepare the QuoteBot
	quotes := je.LoadQuotes(conf.Quotefile)
	ok, conn, reader := je.IrcConnect(&conf)
  if !ok {
		log.Fatalln("Couldn't connect to server")
	}
	ircSend := make(chan je.IrcOperation)

	eppo := je.CreateBot(conf, reader, ircSend, quotes)
	defer conn.Close()

	eppo.TwitterCtl = make(chan string)
	go eppo.ChatContinuous()

	rand.Seed(time.Now().Unix())

	//Prepare the Twitterbot
	twitterSend := make(chan string)
	go func() {
		tb := twitterbot.CreateBot(twitterSend, eppo.TwitterCtl)
		tb.ReadContinuous()
	}()

	send := func(line je.IrcOperation) {
		fmt.Fprint(conn, line.String())
		// If verbose logging is off, just print whatever we say on IRC (except pong)
		if !conf.Verbose && line.Type() != "PONG" {
			log.Print(line.String())
		}
	}

	for {
		select {
		case outLine := <-ircSend:
			send(outLine)
		case outLine := <-twitterSend:
			if conf.Colors {
				outLine = "\x0314" + outLine + "\x0f"
			}
			send(&je.IrcMessage{
				Channel: conf.Channel,
				Text:    outLine,
			})
		}
	}
}
