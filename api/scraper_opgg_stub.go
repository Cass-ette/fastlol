// +build !playwright

package api

// initOPGG returns nil when Playwright is not available
func initOPGG() Scraper {
	return nil
}
