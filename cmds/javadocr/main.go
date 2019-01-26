package main

import (
	"github.com/lukegb/javadocr"
	"github.com/lukegb/javadocr/maven"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {
	u, err := url.Parse("https://repo.spongepowered.org/maven/")
	if err != nil {
		panic(err)
	}

	r := maven.RemoteRepository{URL: u, MayResolveSnapshots: true}
	c := maven.Coordinate{"org.spongepowered", "spongeapi", "", "", ""}
	h, err := javadocr.NewJavadocHandler(r, c)
	if err != nil {
		panic(err)
	}
	h.ExcludeVersion("3.0.1-indev")
	for _, thing := range []string{
		"co", "org",
		"package-list",
		"overview-frame.html",
		"constant-values.html",
		"serialized-form.html",
		"overview-tree.html",
		"index-all.html",
		"deprecated-list.html",
		"allclasses-frame.html",
		"allclasses-noframe.html",
		"index.html",
		"overview-summary.html",
		"help-doc.html",
		"stylesheet.css",
		"script.js",
	} {
		h.AddCompatFor(thing)
	}

	listenOn := os.Getenv("JAVADOCR_LISTEN")
	if listenOn == "" {
		listenOn = ":16080"
	}
	log.Println("ready, listening on", listenOn)
	log.Fatalln(http.ListenAndServe(listenOn, h))
}
