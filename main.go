package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gocarina/gocsv"
)

const (
	// いずれはquery parameterもターゲットにしていきたい
	repoURLFmt      = "https://api.github.com/repos/%s/pulls?state=closed&base=master"
	authHeader      = "Authorization"
	tokenFmt        = "token %s"
	tokenENVKey     = "GITHUB_TOKEN"
	defaultFileName = "out.csv"
)

var location *time.Location

func init() {
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}
	location = jst
}

type dateTime struct {
	time.Time
}

func (d dateTime) MarshalCSV() (string, error) {
	return d.In(location).Format("2006-01-02T15:04:05"), nil
}

type response struct {
	Number               uint       `json:"number" csv:"PR number"`
	MergedAt             dateTime   `json:"merged_at" csv:"master applied at"`
	User                 user       `json:"user" csv:"-"`
	Title                string     `json:"title" csv:"title"`
	HTMLURL              string     `json:"html_url" csv:"URL"`
	Body                 string     `json:"body" csv:"body"`
	ChangeRepresentative string     `csv:"chnage representative"`
	ApprovedAt           *time.Time `csv:"approved at,omitempty"`
}

type user struct {
	Login string `json:"login" csv:"PR authoer"`
}

func (r response) IsTarget() bool {
	return !r.MergedAt.IsZero()
}

func main() {
	out := flag.String("out", defaultFileName, "the file name of the result")
	flag.Parse()
	ownerRepo := flag.Arg(0)
	oFileName := *out

	req, err := http.NewRequest("GET", fmt.Sprintf(repoURLFmt, ownerRepo), nil)
	if err != nil {
		log.Fatal(fmt.Errorf("invalid request: %w", err))
	}
	token := os.Getenv(tokenENVKey)
	req.Header.Set(authHeader, fmt.Sprintf(tokenFmt, token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(fmt.Errorf("request failed: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("request failed status code: %d", resp.StatusCode)
	}

	var res []response
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to decode response: %w", err))
	}

	file, err := os.Create(oFileName)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create a file: %w", err))
	}
	defer file.Close()

	var targets []response
	for i := range res {
		if res[i].IsTarget() {
			targets = append(targets, res[i])
		}
	}

	err = gocsv.MarshalFile(targets, file)
	if err != nil {
		log.Fatal("failed to write a file: %w", err)
	}
}
