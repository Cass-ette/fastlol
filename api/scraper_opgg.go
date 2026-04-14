// +build playwright

package api

import (
	"errors"
	"strings"
)

// initOPGG returns an OPGGScraper if Playwright is available
func initOPGG() Scraper {
	scraper, err := NewOPGGScraper()
	if err != nil {
		// Return a placeholder that will report the error when used
		return &opggUnavailable{err: err}
	}
	return scraper
}

type opggUnavailable struct {
	err error
}

func (o *opggUnavailable) Name() string {
	return "opgg"
}

func (o *opggUnavailable) GetCounters(champion, role string) (*CounterData, error) {
	if strings.Contains(o.err.Error(), "install the driver") {
		return nil, errors.New("Playwright not installed. Run: `go run github.com/playwright-community/playwright-go/cmd/playwright@latest install`")
	}
	return nil, o.err
}

func (o *opggUnavailable) GetMatchup(champion, enemy, role string) (*MatchupResult, error) {
	if strings.Contains(o.err.Error(), "install the driver") {
		return nil, errors.New("Playwright not installed. Run: `go run github.com/playwright-community/playwright-go/cmd/playwright@latest install`")
	}
	return nil, o.err
}
