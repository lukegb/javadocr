package maven

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type SkipResolutionError string

func (s SkipResolutionError) Error() string {
	return string(s)
}

var (
	ErrSnapshotsNotAllowed = SkipResolutionError("Snapshots not resolvable from this repository")
	ErrUnsupportedScheme   = errors.New(`unsupported scheme`)
	ErrBadStatus           = errors.New(`bad HTTP status`)
)

var (
	mavenMetadataURL = &url.URL{
		Path: "maven-metadata.xml",
	}
)

type Repository interface {
	Resolve(Coordinate) (*Artifact, error)
	VersionsForCoordinate(Coordinate) ([]Coordinate, error)
	fetchArtifact(Artifact) (io.ReadCloser, error)
}

type RemoteRepository struct {
	URL                 *url.URL
	MayResolveSnapshots bool
}

func (r RemoteRepository) Resolve(c Coordinate) (*Artifact, error) {
	if !r.MayResolveSnapshots && c.IsSnapshot() {
		return nil, ErrSnapshotsNotAllowed
	}

	cdurl, err := r.coordinateDirectoryURL(c)
	if err != nil {
		return nil, err
	}

	var mm *MavenMetadata
	if c.IsSnapshot() {
		// we need to narrow down which version we're actually talking about
		// and then we can ask the coordinate for a final filename

		// this means we need to retrieve the maven-metadata.xml for this
		// directory, so here we go...!
		mmurl := cdurl.ResolveReference(mavenMetadataURL)
		rc, err := r.get(mmurl)
		if err != nil {
			return nil, err
		}

		mm, err = parseMavenMetadata(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
	}

	filename, err := c.filename(mm)
	if err != nil {
		return nil, err
	}

	aurl := cdurl.ResolveReference(&url.URL{
		Path: filename,
	})

	return &Artifact{
		Coordinate: c,
		URL:        aurl,
		repository: r,
	}, nil
}

func (r RemoteRepository) fetchArtifact(a Artifact) (io.ReadCloser, error) {
	return r.get(a.URL)
}

func (r RemoteRepository) get(u *url.URL) (io.ReadCloser, error) {
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrUnsupportedScheme
	}

	req := &http.Request{
		Method: "GET",
		URL:    u,
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, ErrBadStatus
	}
	return resp.Body, nil
}

func (r RemoteRepository) coordinateDirectory(c Coordinate) string {
	return path.Join(
		append(strings.Split(c.GroupId, "."),
			c.ArtifactId,
			c.Version,
		)...,
	) + "/" /* force a trailing slash */
}

func (r RemoteRepository) coordinateDirectoryURL(c Coordinate) (*url.URL, error) {
	cdurl, err := url.Parse(r.coordinateDirectory(c))
	if err != nil {
		return nil, err
	}

	return r.URL.ResolveReference(cdurl), nil
}

func (r RemoteRepository) VersionsForCoordinate(c Coordinate) ([]Coordinate, error) {
	// this is sort of cheating - we take a coordinate as input and produce several more
	c.Version = ""
	cdurl, err := r.coordinateDirectoryURL(c)
	if err != nil {
		return nil, err
	}

	mmurl := cdurl.ResolveReference(mavenMetadataURL)
	rc, err := r.get(mmurl)
	if err != nil {
		return nil, err
	}

	mm, err := parseMavenMetadata(rc)
	rc.Close()
	if err != nil {
		return nil, err
	}

	coords := make([]Coordinate, len(mm.Versioning.Versions))
	for n, v := range mm.Versioning.Versions {
		c.Version = v
		coords[n] = c
	}
	return coords, nil
}
