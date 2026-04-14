// +build playwright

package api

import (
	"fmt"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// OPGGScraper uses Playwright to scrape OP.GG (requires Chromium)
type OPGGScraper struct {
	pw      *playwright.Playwright
	browser playwright.Browser
}

func NewOPGGScraper() (*OPGGScraper, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("launch browser: %w", err)
	}

	return &OPGGScraper{
		pw:      pw,
		browser: browser,
	}, nil
}

func (s *OPGGScraper) Name() string {
	return "opgg"
}

func (s *OPGGScraper) Close() error {
	if s.browser != nil {
		s.browser.Close()
	}
	if s.pw != nil {
		s.pw.Stop()
	}
	return nil
}

func (s *OPGGScraper) GetCounters(champion, role string) (*CounterData, error) {
	url := fmt.Sprintf("https://www.op.gg/champions/%s/counters", strings.ToLower(champion))

	page, err := s.browser.NewPage(playwright.BrowserNewPageOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return nil, fmt.Errorf("new page: %w", err)
	}
	defer page.Close()

	// Set up console log capture
	page.On("console", func(msg playwright.ConsoleMessage) {
		fmt.Fprintf(os.Stderr, "[Browser Console] %s\n", msg.Text())
	})

	if _, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); err != nil {
		return nil, fmt.Errorf("goto: %w", err)
	}

	// Wait for content
	page.WaitForTimeout(5000)

	// Debug: Get page title and URL
	title, _ := page.Title()
	url2 := page.URL()
	fmt.Fprintf(os.Stderr, "Page title: %s\n", title)
	fmt.Fprintf(os.Stderr, "Current URL: %s\n", url2)

	// Debug: Save innerHTML to file
	innerHTML, _ := page.Evaluate(`() => document.body.innerHTML.substring(0, 10000)`, nil)
	if html, ok := innerHTML.(string); ok {
		fmt.Fprintf(os.Stderr, "Body preview: %s...\n", html[:min(500, len(html))])
	}

	// Try to find percentage data with a comprehensive script
	script := `
		() => {
			const result = {
				tables: 0,
				rows: 0,
				percentages: [],
				dataWinRate: 0,
				textNodes: 0
			};

			// Count tables
			result.tables = document.querySelectorAll('table').length;

			// Count table rows
			result.rows = document.querySelectorAll('table tbody tr').length;

			// Count data-win-rate attributes
			result.dataWinRate = document.querySelectorAll('[data-win-rate]').length;

			// Find percentage text nodes using TreeWalker
			const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
			let node;
			while (node = walker.nextNode()) {
				const text = (node.textContent || '').trim();
				if (/^\d{2,3}\.\d%$/.test(text)) {
					result.percentages.push({
						text,
						parent: node.parentElement?.tagName,
						parentClass: node.parentElement?.className?.substring(0, 50)
					});
					result.textNodes++;
				}
			}

			// Find any text containing percentages
			const allText = [];
			document.querySelectorAll('*').forEach(el => {
				const text = el.textContent || '';
				if (/\d{2,3}\.\d%/.test(text)) {
					allText.push({
						tag: el.tagName,
						class: el.className?.substring(0, 50),
						text: text.trim().substring(0, 100)
					});
				}
			});

			return { ...result, allText: allText.slice(0, 10) };
		}
	`

	var debugResult interface{}
	page.Evaluate(script, &debugResult)
	fmt.Fprintf(os.Stderr, "Debug result: %+v\n", debugResult)

	// Now extract actual data
	data := &CounterData{
		Champion: champion,
		Role:     role,
		Source:   s.Name(),
	}

	// Try to get matchups
	matchupScript := `
		() => {
			const matchups = [];

			document.querySelectorAll('table tbody tr').forEach(row => {
				const cells = row.querySelectorAll('td');
				if (cells.length < 2) return;

				let wr = 0;
				let name = '';
				let games = 0;

				cells.forEach(cell => {
					const text = cell.textContent.trim();

					// Check for win rate
					const wrMatch = text.match(/(\d{2,3}\.\d)%/);
					if (wrMatch) {
						wr = parseFloat(wrMatch[1]);
					}

					// Check for games
					const gamesMatch = text.match(/([\d,]+)\s*(games?|场)/i);
					if (gamesMatch) {
						games = parseInt(gamesMatch[1].replace(/,/g, ''));
					}

					// Get champion name
					if (!name) {
						const link = cell.querySelector('a[href*="champion"]');
						const img = cell.querySelector('img');
						if (link) {
							name = link.textContent.trim();
						} else if (img && img.alt) {
							name = img.alt.trim();
						} else if (text.length > 1 && text.length < 30 && /^[A-Za-z]/.test(text)) {
							name = text.split(/\n/)[0].trim();
						}
					}
				});

				if (name && name.length > 1 && name.length < 30 && wr > 0) {
					matchups.push({ name, winRate: wr, games });
				}
			});

			return matchups;
		}
	`

	var matchups []interface{}
	page.Evaluate(matchupScript, &matchups)
	fmt.Fprintf(os.Stderr, "Found %d matchups from table\n", len(matchups))

	// Sort and split
	if len(matchups) > 0 {
		// Convert and sort
		type matchup struct {
			Name    string
			WinRate float64
			Games   int
		}

		var parsed []matchup
		for _, m := range matchups {
			if mm, ok := m.(map[string]interface{}); ok {
				name, _ := mm["name"].(string)
				wr, _ := mm["winRate"].(float64)
				games, _ := mm["games"].(float64)
				if name != "" && wr > 0 {
					parsed = append(parsed, matchup{Name: name, WinRate: wr, Games: int(games)})
				}
			}
		}

		// Simple bubble sort
		for i := 0; i < len(parsed); i++ {
			for j := i + 1; j < len(parsed); j++ {
				if parsed[j].WinRate < parsed[i].WinRate {
					parsed[i], parsed[j] = parsed[j], parsed[i]
				}
			}
		}

		mid := min(5, len(parsed)/2)
		for i := 0; i < mid && i < len(parsed); i++ {
			data.WeakAgainst = append(data.WeakAgainst, Matchup{
				Name:    parsed[i].Name,
				WinRate: parsed[i].WinRate / 100,
				Games:   parsed[i].Games,
			})
		}
		for i := len(parsed) - mid; i < len(parsed) && i >= 0; i++ {
			data.StrongAgainst = append(data.StrongAgainst, Matchup{
				Name:    parsed[i].Name,
				WinRate: parsed[i].WinRate / 100,
				Games:   parsed[i].Games,
			})
		}
	}

	return data, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeOPGGRole(role string) string {
	role = strings.ToLower(role)
	switch role {
	case "top":
		return "top"
	case "jungle", "jg", "jun":
		return "jungle"
	case "mid", "middle":
		return "mid"
	case "adc", "bot", "bottom":
		return "adc"
	case "support", "sup":
		return "support"
	default:
		return "top"
	}
}

// GetMatchup gets the specific 1v1 matchup win rate between two champions
func (s *OPGGScraper) GetMatchup(champion, enemy, role string) (*MatchupResult, error) {
	normalizedRole := normalizeOPGGRole(role)
	url := fmt.Sprintf("https://www.op.gg/champions/%s/counters/%s?target_champion=%s",
		strings.ToLower(champion), normalizedRole, strings.ToLower(enemy))

	page, err := s.browser.NewPage(playwright.BrowserNewPageOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
	})
	if err != nil {
		return nil, fmt.Errorf("new page: %w", err)
	}
	defer page.Close()

	if _, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); err != nil {
		return nil, fmt.Errorf("goto: %w", err)
	}

	page.WaitForTimeout(5000)

	script := `
		() => {
			let winRate = 0;

			document.querySelectorAll('table tbody tr').forEach(row => {
				const text = row.textContent;
				const match = text.match(/(\d{2,3}\.\d)%/);
				if (match && !winRate) {
					winRate = parseFloat(match[1]);
				}
			});

			if (!winRate) {
				document.querySelectorAll('[data-win-rate]').forEach(el => {
					if (!winRate) {
						winRate = parseFloat(el.getAttribute('data-win-rate'));
					}
				});
			}

			if (!winRate) {
				const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
				let node;
				while (node = walker.nextNode()) {
					const text = node.textContent || '';
					if (/^\d{2,3}\.\d%$/.test(text.trim())) {
						winRate = parseFloat(text.trim().replace('%', ''));
						break;
					}
				}
			}

			return { winRate };
		}
	`

	var result map[string]interface{}
	page.Evaluate(script, &result)
	winRate, _ := result["winRate"].(float64)

	if winRate == 0 {
		return nil, fmt.Errorf("win rate not found")
	}

	return &MatchupResult{
		Champion: champion,
		Enemy:    enemy,
		WinRate:  winRate / 100,
		Source:   s.Name(),
	}, nil
}
