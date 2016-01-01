package main

import (
	"github.com/lukegb/javadocr"
	"github.com/lukegb/javadocr/maven"
	"log"
	"net/http"
	"net/url"
)

func main() {
	u, err := url.Parse("http://repo.spongepowered.org/maven/")
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

	log.Println("ready")
	log.Fatalln(http.ListenAndServe(":8181", h))
}
