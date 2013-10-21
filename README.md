Installation
============

Several non-standard packages must be installed for this to work. To get them,

	go get "code.google.com/p/gcfg"
	go get "github.com/mrjones/oauth"
	go get "code.google.com/p/go.net/html"

To compile JanEppo, you need `net/html`, which is currently in development and not in the main distro.

Now if you're on my machine, go get will download this, but then complain about `io.ErrNoProgress` not existing. If you're like me, you do not feel like solving this problem the elegant way and getting all kinds of funky bleeding-edge versions of packages. The alternative is to edit

	[/usr/lib/go]/src/pkg/code.google.com/p/go.net/html/token.go

The part between brackets may be different on your machine, it's `GOROOT` as returned by `go env`.  You can search for `io.ErrNoProgress` and replace it by `nil` (or a more sensible value you have thought about). This will work fine for our purposes.

When you are done, you should be able to build the project; `html` will be compiled as part of it and (hopefully) work as intended.

Configuration
=============

You need to supply your own versions of some config files.

collega.json
------------

This is the list of quotes. It's in JSON format, and might look like this:

	[{"Name":"Erik","Text":"Hello"},{"Name":"Fred","Text":"Bye"}]

More quotes will make for a better bot. You can add quotes from within the bot too, but the file must exist.

twitter.json
------------
Needs to contain your own app authentication for oAuth and users to follow on Twitter. Made mostly automatically by the accompanying program, but a skeleton file should be provided, and you should probably follow at least one person. An example is given below.

	{"CnsKey":"",
	 "CnsSecret":"",
	 "Follow":"12345",
	 "AccessToken":{"Token":"","Secret":""}}

Available commands
==================
- `!collega [Query]`
    Displays a random quote from the database, or optionally, one by a person matching the query.
    `!janeppo` is short for `!collega janeppo`.
- `!wiezei Query`
    Displays a quote with query in the message
- `!watzei Someone over Something
    Combination of the above two commands
- `!addquote Someone: Something`
    Adds a quote to the database.
- `!undo`
    Removes the quote that was last added.
- `!college`, `!collage`
    Misspellings of `!collega` that lead to a bogus response
- `gang`, `LAZER`
    Typing these will lead to an echo.
- `!sikknel`
    Reads information from a scanner of the emergency service comms service and prints it. Useful for finding out where the fire truck was headed that just passed your house.
- `!waaris Query`
    Prints RUG building information matching Query.
- `Botname: verdwijn`
    Causes the bot to immediately quit. Use in case of nasty bugs clogging the channel.
- `!raw Command`
    Allows for sending raw IRC commands as the bot. Use with care.
- `!ops`
    Request ops. The bot will attempt to comply, but if it's not an op, it won't work.
- `!fixtwitter`, `!follow Username`, `!unfollow Username`, `!following`, `!link`
    Various commands to control the twitter functionality. The first command resets the twitter connection, the last one posts a link to the last tweet.

Apart from these, the bot contains various joke commands and a link shortener.