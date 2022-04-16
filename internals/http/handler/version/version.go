package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var (
	Name      = "DEVELOP_BUILD"
	Version   = "DEVELOP_VERSION"
	BuildTime = "NO_BUILD_TIME"
	Commit    = "NO_COMMIT"
	Branch    = "NO_BRANCH"
	StartTime time.Time
)

func init() {
	StartTime = time.Now().UTC()
}

// Payload is the Version information
type Payload struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	BuildTime string    `json:"build_time"`
	StartTime time.Time `json:"start_time"`
	Commit    string    `json:"commit"`
	Branch    string    `json:"branch"`
}

// Handler returns a new version handler
func Handler() http.Handler {
	return CustomHandler(Payload{
		Name:      Name,
		Version:   Version,
		BuildTime: BuildTime,
		StartTime: StartTime,
		Commit:    Commit,
		Branch:    Branch,
	})
}

// CustomHandler returns a new version handler
func CustomHandler(pl Payload) http.Handler {
	if pl.StartTime.IsZero() {
		pl.StartTime = time.Now().UTC()
	}
	var s string
	b, err := json.Marshal(pl)
	s = string(b)
	if err != nil {
		s = "JSON Error: " + err.Error()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, s)
	})
}
