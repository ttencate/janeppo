package eppobot

import (
	"regexp"
)

type handler func(*QuoteBot, *IrcMessage, []string)

type ActionHandler struct {
	Regexp  *regexp.Regexp
	Handler handler
}

var messageToAction []ActionHandler = []ActionHandler{
	// Panic handler
	ActionHandler{regexp.MustCompile("^(\\w+): verdwijn"), forceDisconnect},
	// Handlers for QDB-related queries (read)
	ActionHandler{regexp.MustCompile("^!collega$"), sayQuote},
	ActionHandler{regexp.MustCompile("^!collega (.+)$"), sayQuote},
	ActionHandler{regexp.MustCompile("^!wiezei( )(.+)$"), sayQuote},
	ActionHandler{regexp.MustCompile("^!watzei (.+) over (.+)$"), sayQuote},
	ActionHandler{regexp.MustCompile("^!college$"), respondCollege},
	ActionHandler{regexp.MustCompile("^!collage$"), reverseQuote},
	ActionHandler{regexp.MustCompile("^!janeppo$"), selfQuote},
	// (write)
	ActionHandler{regexp.MustCompile("^!addquote ([^:]+): (.+)$"), addQuote},
	ActionHandler{regexp.MustCompile("^!undo$"), undoAddQuote},
	ActionHandler{regexp.MustCompile("^!herlaad$"), reloadDatabase},
	// Random nonsense
	ActionHandler{regexp.MustCompile("^!pikk$"), measureAttachment},
	ActionHandler{regexp.MustCompile("^!ijbepikk$"), measureFrustration},
	ActionHandler{regexp.MustCompile("^gang"), simpleResponder("GANG!!!")},
	ActionHandler{regexp.MustCompile("(?i)^lazer"), simpleResponder("LAZERS!")},
	ActionHandler{regexp.MustCompile("^!sl$"), train},
	// Lookup services
	ActionHandler{regexp.MustCompile("^!sikknel$"), dispatchP2k},
	ActionHandler{regexp.MustCompile("^!waaris (.+)$"), findBuilding},
	ActionHandler{regexp.MustCompile("http"), shortenLink},
	// Bot controls
	ActionHandler{regexp.MustCompile("^!raw ([^ ]+) (.+)$"), rawCommand},
	ActionHandler{regexp.MustCompile("^!ops$"), giveOps},
	// Twitterbot controls
	ActionHandler{regexp.MustCompile("^!fixtwitter$"), twitterReset},
	ActionHandler{regexp.MustCompile("^!follow (.+)$"), twitterAdd},
	ActionHandler{regexp.MustCompile("^!unfollow (.+)$"), twitterRem},
	ActionHandler{regexp.MustCompile("^!following$"), twitterList},
	ActionHandler{regexp.MustCompile("^!link( (.+))?$"), twitterLink},
	// Generic response
	ActionHandler{regexp.MustCompile("^(\\w+): "), genericResponse},
}

func simpleResponder(s string) handler {
	return func(b *QuoteBot, in *IrcMessage, submatches []string) {
		b.Output <- &IrcMessage{
			Channel: in.Channel,
			Text:    s,
		}
	}
}
