package maven

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidCoordinate        = errors.New(`invalid coordinate`)
	ErrSnapshotRequiresMetadata = errors.New(`snapshot resolution requires passing metadata`)
)

type Coordinate struct {
	GroupId    string
	ArtifactId string
	Packaging  string
	Classifier string
	Version    string
}

func (c Coordinate) String() string {
	var arr []string
	if c.Packaging != "" && c.Classifier != "" {
		arr = []string{
			c.GroupId, c.ArtifactId, c.Packaging, c.Classifier, c.Version,
		}
	} else if c.Packaging != "" {
		arr = []string{
			c.GroupId, c.ArtifactId, c.Packaging, c.Version,
		}
	} else {
		arr = []string{
			c.GroupId, c.ArtifactId, c.Version,
		}
	}
	return strings.Join(arr, ":")
}

func CoordinateFromString(s string) (Coordinate, error) {
	arr := strings.Split(s, ":")
	if len(arr) < 3 || len(arr) > 5 {
		return Coordinate{}, ErrInvalidCoordinate
	}

	if len(arr) == 5 {
		return Coordinate{
			arr[0], arr[1], arr[2], arr[3], arr[4],
		}, nil
	} else if len(arr) == 4 {
		return Coordinate{
			arr[0], arr[1], arr[2], "", arr[3],
		}, nil
	}

	return Coordinate{
		arr[0], arr[1], "", "", arr[2],
	}, nil
}

func (c Coordinate) IsSnapshot() bool {
	return strings.HasSuffix(c.Version, "-SNAPSHOT")
}

func (c Coordinate) filename(mm *MavenMetadata) (string, error) {
	if c.IsSnapshot() && mm == nil {
		return "", ErrSnapshotRequiresMetadata
	}

	ver := c.Version
	if c.IsSnapshot() {
		ver = strings.Replace(c.Version, "-SNAPSHOT", fmt.Sprintf("-%s-%d", mm.Versioning.Snapshot.Timestamp, mm.Versioning.Snapshot.BuildNumber), 1)
	}

	packaging := "jar"
	if c.Packaging != "" {
		packaging = c.Packaging
	}

	classifier := ""
	if c.Classifier != "" {
		classifier = "-" + c.Classifier
	}

	return fmt.Sprintf("%s-%s%s.%s", c.ArtifactId, ver, classifier, packaging), nil
}
