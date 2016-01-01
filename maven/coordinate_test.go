package maven

import (
	"testing"
)

var TestValidExamples = map[string]Coordinate{
	"org.spongepowered:spongeapi:3.0.0": Coordinate{
		"org.spongepowered", "spongeapi", "", "", "3.0.0",
	},
	"org.spongepowered:spongeapi:pom:3.0.0": Coordinate{
		"org.spongepowered", "spongeapi", "pom", "", "3.0.0",
	},
	"org.spongepowered:spongeapi:jar:javadoc:3.0.0": Coordinate{
		"org.spongepowered", "spongeapi", "jar", "javadoc", "3.0.0",
	},
}

func TestCoordParse(t *testing.T) {
	for in, out := range TestValidExamples {
		coord, err := CoordinateFromString(in)
		if err != nil {
			t.Errorf("CoordinateFromString(\"%s\") returned error: %s", coord, err)
		} else if coord != out {
			t.Errorf("Got: %#v, expected: %#v", coord, out)
		}
	}
}

func TestCoordString(t *testing.T) {
	for out, in := range TestValidExamples {
		coordstr := in.String()
		if coordstr != out {
			t.Errorf("Got: %s, expected: %s", coordstr, out)
		}
	}
}
