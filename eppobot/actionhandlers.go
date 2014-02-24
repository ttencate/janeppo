package eppobot

import (
	"regexp"
)

type ActionHandler func(*QuoteBot, *IrcMessage, []string)

var messageToAction map[*regexp.Regexp]ActionHandler = map[*regexp.Regexp]ActionHandler{
	// Handlers for QDB-related queries (read)
	regexp.MustCompile("^!collega$"):               sayQuote,
	regexp.MustCompile("^!collega (.*)$"):          sayQuote,
	regexp.MustCompile("^!wiezei( )(.*)$"):         sayQuote,
	regexp.MustCompile("^!watzei (.*) over (.*)$"): sayQuote,
	regexp.MustCompile("^!college$"):               respondCollege,
	regexp.MustCompile("^!collage$"):               reverseQuote,
	regexp.MustCompile("^!janeppo$"):               selfQuote,
	// (write)
	regexp.MustCompile("^!addquote ([^:]*): (.*)$"): addQuote,
	regexp.MustCompile("^!undo$"):                   undoAddQuote,
	regexp.MustCompile("^!herlaad$"):                reloadDatabase,
	// Random nonsense
	regexp.MustCompile("^!pikk$"):     measureAttachment,
	regexp.MustCompile("^!ijbepikk$"): measureFrustration,
	regexp.MustCompile("^gang"):       simpleResponder("GANG!!!"),
	regexp.MustCompile("(?i)^lazer"):  simpleResponder("LAZERS!"),
	regexp.MustCompile("^!sl$"):       train,
	// Lookup services
	regexp.MustCompile("^!sikknel$"):     dispatchP2k,
	regexp.MustCompile("^!waaris (.*)$"): findBuilding,
	regexp.MustCompile("http"):           shortenLink,
	// Bot controls
	regexp.MustCompile("^[^ ]*: verdwijn"):    forceDisconnect,
	regexp.MustCompile("^!raw ([^ ]*) (.*)$"): rawCommand,
	regexp.MustCompile("^!ops$"):              giveOps,
	// Twitterbot controls
	regexp.MustCompile("^!fixtwitter$"):    twitterReset,
	regexp.MustCompile("^!follow (.*)$"):   twitterAdd,
	regexp.MustCompile("^!unfollow (.*)$"): twitterRem,
	regexp.MustCompile("^!following$"):     twitterList,
	regexp.MustCompile("^!link( (.*))?$"):  twitterLink,
}

func simpleResponder(s string) ActionHandler {
	return func(b *QuoteBot, in *IrcMessage, submatches []string) {
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    s,
		}
	}
}
