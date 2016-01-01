package maven

import (
	"encoding/xml"
	"io"
)

type MavenMetadata struct {
	XMLName    xml.Name `xml:"metadata"`
	GroupId    string   `xml:"groupId"`
	ArtifactId string   `xml:"artifactId"`
	Version    string   `xml:"version"`
	Versioning struct {
		Snapshot struct {
			Timestamp   string `xml:"timestamp"`
			BuildNumber int    `xml:"buildNumber"`
		} `xml:"snapshot"`
		Versions    []string `xml:"versions>version"`
		LastUpdated string   `xml:"lastUpdated"`
		Release     string   `xml:"release"`
	} `xml:"versioning"`
}

func parseMavenMetadata(r io.Reader) (*MavenMetadata, error) {
	mm := new(MavenMetadata)
	d := xml.NewDecoder(r)
	err := d.Decode(&mm)
	return mm, err
}
