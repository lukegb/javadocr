package maven

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRepositoryDirectoryBuilder(t *testing.T) {
	testPlan := map[Coordinate]string{
		Coordinate{
			"org.spongepowered", "spongeapi", "", "", "3.0.0",
		}: "org/spongepowered/spongeapi/3.0.0/",
		Coordinate{
			"org.spongepowered", "spongeapi", "", "", "2.1-SNAPSHOT",
		}: "org/spongepowered/spongeapi/2.1-SNAPSHOT/",
	}
	rr := RemoteRepository{}
	for coord, out := range testPlan {
		res := rr.coordinateDirectory(coord)
		if res != out {
			t.Errorf("Got: %s, expected: %s", res, out)
		}
	}
}

func testRepositoryResolution(t *testing.T, rr RemoteRepository, expectSnapshotFailure bool) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "%s", `<metadata>
<groupId>org.spongepowered</groupId>
<artifactId>spongeapi</artifactId>
<version>2.1-SNAPSHOT</version>
<versioning>
<snapshot>
<timestamp>20160101.061445</timestamp>
<buildNumber>272</buildNumber>
</snapshot>
<lastUpdated>20160101061445</lastUpdated>
</versioning>
</metadata>`)
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	rr.URL = u

	coordOrPanic := func(c string) Coordinate {
		coord, err := CoordinateFromString(c)
		if err != nil {
			t.Fatal(err)
		}
		return coord
	}

	coordinates := map[Coordinate]string{
		coordOrPanic("org.spongepowered:spongeapi:3.0.0"):             "/org/spongepowered/spongeapi/3.0.0/spongeapi-3.0.0.jar",
		coordOrPanic("org.spongepowered:spongeapi:pom:3.0.0"):         "/org/spongepowered/spongeapi/3.0.0/spongeapi-3.0.0.pom",
		coordOrPanic("org.spongepowered:spongeapi:jar:javadoc:3.0.0"): "/org/spongepowered/spongeapi/3.0.0/spongeapi-3.0.0-javadoc.jar",
	}
	for coord, dest := range coordinates {
		artifact, err := rr.Resolve(coord)
		if err != nil {
			t.Error(err)
			continue
		}
		if artifact.URL.Path != dest {
			t.Errorf("got: %s, expected: %s", artifact.URL.Path, dest)
		}
	}

	snapshotCoordinates := map[Coordinate]string{
		coordOrPanic("org.spongepowered:spongeapi:2.1-SNAPSHOT"):             "/org/spongepowered/spongeapi/2.1-SNAPSHOT/spongeapi-2.1-20160101.061445-272.jar",
		coordOrPanic("org.spongepowered:spongeapi:pom:2.1-SNAPSHOT"):         "/org/spongepowered/spongeapi/2.1-SNAPSHOT/spongeapi-2.1-20160101.061445-272.pom",
		coordOrPanic("org.spongepowered:spongeapi:jar:javadoc:2.1-SNAPSHOT"): "/org/spongepowered/spongeapi/2.1-SNAPSHOT/spongeapi-2.1-20160101.061445-272-javadoc.jar",
	}
	for coord, dest := range snapshotCoordinates {
		artifact, err := rr.Resolve(coord)
		if expectSnapshotFailure && err != ErrSnapshotsNotAllowed {
			t.Error("expected ErrSnapshotsNotAllowed, got %#v", err)
			continue
		} else if expectSnapshotFailure {
			continue
		}
		if err != nil {
			t.Error(err)
			continue
		}
		if artifact.URL.Path != dest {
			t.Errorf("got: %s, expected: %s", artifact.URL.Path, dest)
		}
	}
}

func TestRepositoryResolution(t *testing.T) {
	snapshotr := RemoteRepository{
		MayResolveSnapshots: true,
	}
	releaser := RemoteRepository{
		MayResolveSnapshots: false,
	}

	testRepositoryResolution(t, snapshotr, false)
	testRepositoryResolution(t, releaser, true)
}

func TestRepositoryVersionsForCoordinate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/org/spongepowered/spongeapi/maven-metadata.xml" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		fmt.Fprintln(w, "%s", `
<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <groupId>org.spongepowered</groupId>
  <artifactId>spongeapi</artifactId>
  <versioning>
    <release>3.0.1-indev</release>
    <versions>
      <version>1.0.0-SNAPSHOT</version>
      <version>1.0</version>
      <version>1.1-SNAPSHOT</version>
      <version>2.0</version>
      <version>2.1-SNAPSHOT</version>
      <version>3.0.0</version>
      <version>3.0.1-indev</version>
    </versions>
    <lastUpdated>20160101075640</lastUpdated>
  </versioning>
</metadata>`)
	}))
	defer ts.Close()

	coords := []Coordinate{
		{"org.spongepowered", "spongeapi", "", "", "1.0.0-SNAPSHOT"},
		{"org.spongepowered", "spongeapi", "", "", "1.0"},
		{"org.spongepowered", "spongeapi", "", "", "1.1-SNAPSHOT"},
		{"org.spongepowered", "spongeapi", "", "", "2.0"},
		{"org.spongepowered", "spongeapi", "", "", "2.1-SNAPSHOT"},
		{"org.spongepowered", "spongeapi", "", "", "3.0.0"},
		{"org.spongepowered", "spongeapi", "", "", "3.0.1-indev"},
	}

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	rr := RemoteRepository{
		URL:                 u,
		MayResolveSnapshots: true,
	}
	vers, err := rr.VersionsForCoordinate(Coordinate{"org.spongepowered", "spongeapi", "", "", "3.0.1-indev"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vers) != len(coords) {
		t.Fatalf("got %d elements, expected %d", len(vers), len(coords))
	}
	for n := range vers {
		if vers[n] != coords[n] {
			t.Fatalf("in position %d, got %#v, expected %#v", n, vers[n], coords[n])
		}
	}
}
