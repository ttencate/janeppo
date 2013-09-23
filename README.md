Installation
============

Several non-standard packages must be installed for this to work. Using `go get`, install these packages:
	"code.google.com/p/gcfg"
	"code.google.com/p/go.net/html"
	"github.com/mrjones/oauth"

To compile JanEppo, you need net/html, which is currently in development and not
in the main distro.

To get it, 
	go get "code.google.com/p/go.net/html"

Now if you're on my machine, go get will download this, but then complain about
io.ErrNoProgress not existing. If you're like me, you do not feel like solving
this problem the elegant way and getting all kinds of funky bleeding-edge
versions of packages. The alternative is to edit
	[/usr/lib/go]/src/pkg/code.google.com/p/go.net/html/token.go
The part between brackets may be different on your machine, it's GOROOT as
returned by `go env'.  You can search for io.ErrNoProgress and replace it by nil
(or a more sensible value you have thought about). This will work fine for our
purposes.

When you are done, you should be able to build the project; html will be
compiled as part of it and (hopefully) work as intended.

Configuration
=============

You need to supply your own versions of some config files.

collega.txt
-----------

This is the list of quotes. It's in JSON format, and might look like this:
	[{"Name":"Erik","Text":"Hello"},{"Name":"Fred","Text":"Bye"}]

More quotes will make for a better bot. You can add quotes from within the bot too, but the file must exist.

twitter.ini
-----------
Needs to contain your own app authentication for oAuth and users to follow on Twitter. A typical file will be
	[oauth]
	CnsKey = (consumer key goes here)
	CnsSecret = (consumer secret goes here)
	[twitter]
	follow = 123,456 ; comma-separated list of user ids
