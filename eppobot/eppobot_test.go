package eppobot

import (
	"testing"
	"strings"
	"fmt"
	"math/rand"
	"bufio"
)

// Return a new bot for testing. It has some quotes but no reader.
// To make the bot react to a command, provide it lines of IRC chatter
// by attaching a strings.Reader to it and calling ChatLine.
func initDummyBot() *QuoteBot {
	qdb := []Quote{
		Quote{"Erik", "This is a test"},
		Quote{"Harm", "Let's be honest - almost right is the same as completely wrong."},
		Quote{"Mark", "There's a new LaTeX-reader this year!"},
	}
	conf := Config{
		Nickname:  "TestBot",
		Server:    "localhost",
		Channel:   "#bottest",
		Quotefile: "",
		UrlLength: 100000,
		AutoOps:   true,
		Verbose:   false,
		Colors:    true,
	}
	return &QuoteBot{
		Config:     conf,
		Qdb:        qdb,
		Reader:     nil,
		Output:     make(chan IrcOperation),
		InitLen:    len(qdb),
		TwitterCtl: make(chan string),
	}
}

func (b *QuoteBot) response(command string) IrcOperation {
	b.Reader = bufio.NewReader(strings.NewReader(command + "\n"))
	go b.ChatLine()
	return <-b.Output
}

// This is the common code for many tests
func (b *QuoteBot) chatResponse(message string) IrcOperation {
	return b.response(fmt.Sprintf("someone!somewhere PRIVMSG %s :%s", b.Channel, message))
}

func TestCollega(test *testing.T) {
	b := initDummyBot()
	rand.Seed(2)

	resps := b.chatResponse("!collega")
	if resps.String() != fmt.Sprintf("PRIVMSG %s :Mijn collega %s zou zeggen: \"%s\"\n", b.Channel, b.Qdb[1].Name, b.Qdb[1].Text) {
		test.Error("Failed collega1 with", resps.String())
	}

	resps = b.chatResponse("!collega Erik")
	if resps.String() != fmt.Sprintf("PRIVMSG %s :Mijn collega %s zou zeggen: \"%s\"\n", b.Channel, b.Qdb[0].Name, b.Qdb[0].Text) {
		test.Error("Failed collega2 with", resps.String())
	}

	resps = b.chatResponse("!wiezei right")
	if resps.String() != fmt.Sprintf("PRIVMSG %s :Mijn collega %s zou inderdaad zeggen: \"%s\"\n", b.Channel, b.Qdb[1].Name, b.Qdb[1].Text) {
		test.Error("Failed wiezei with", resps.String())
	}

	resps = b.chatResponse("!watzei ar over eX")
	if resps.String() != fmt.Sprintf("PRIVMSG %s :Mijn collega %s zou inderdaad zeggen: \"%s\"\n", b.Channel, b.Qdb[2].Name, b.Qdb[2].Text) {
		test.Error("Failed watzei with", resps.String())
	}
}

func TestPanic(test *testing.T) {
	b := initDummyBot()
	defer func() {
		if r := recover(); r == nil {
			test.Fail()
		}
	}()

	// This should panic the bot (safety valve)
	b.Reader = bufio.NewReader(strings.NewReader(fmt.Sprintf("someone!somewhere PRIVMSG %s :%s\n", b.Channel, b.Nickname + ": verdwijn")))
	go func() {
		test.Log(<-b.Output)
	}()
	b.ChatLine()
}

func TestSikknel(test *testing.T) {
	b := initDummyBot()
	resps := b.chatResponse("!sikknel")
	if !strings.Contains(resps.String(), fmt.Sprintf("PRIVMSG %s :P", b.Channel)) {
		test.Error("Sikknel returns >", resps.String(), "< but expected >", fmt.Sprintf("PRIVMSG %s :P", b.Channel), "<")
	}
}
