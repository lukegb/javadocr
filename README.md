# javadocr
Serve Javadocs out of Maven repositories ([example](https://jd.spongepowered.org))

## Usage
```
go get github.com/lukegb/javadocr/cmds/javadocr
go build github.com/lukegb/javadocr/cmds/javadocr
./javadocr
```

## Customising
At the moment the command will only serve the [SpongeAPI](https://github.com/SpongePowered/SpongeAPI) documentation.
You can edit this by changing the code. Sorry.

Other things you can't yet customise: the expiry time for SNAPSHOT artifacts. Release artifacts are cached
indefinitely but will be expired if memory usage crosses a (yet again, hardcoded) threshold.

It will, by default, serve on port `16080` on all interfaces, but you can set `JAVADOCR_LISTEN`
to a golang-listen string (ala `:16080` or `127.0.0.1:8181`) to listen elsewhere.

## URLs
The URL scheme is:

http://listeningat/mavenversion/<path to docs>

## How?
It periodically fetches the available versions of a particular project from a Maven repository. It then
allows requests for these versions, at which point it looks up the URL of the javadoc artifact (which must
be in the repo), and then serves them.

## Why?
This was written for the [SpongePowered](https://www.spongepowered.org) project
to power [jd.spongepowered.org](https://jd.spongepowered.org).
