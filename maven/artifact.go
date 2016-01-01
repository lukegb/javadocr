package maven

import (
	"io"
	"net/url"
)

type Artifact struct {
	Coordinate Coordinate
	URL        *url.URL

	repository Repository
}

func (a Artifact) Fetch() (io.ReadCloser, error) {
	return a.repository.fetchArtifact(a)
}
