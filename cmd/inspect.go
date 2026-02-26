package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/the20100/g-search-console-cli/internal/output"
	gsc "google.golang.org/api/searchconsole/v1"
)

var (
	inspectSite     string
	inspectLanguage string
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <url>",
	Short: "Inspect URL indexing status in Google Search",
	Long: `Inspect the indexing status, coverage, mobile usability, and rich results for a URL.

Examples:
  gsc inspect https://example.com/page --site https://example.com
  gsc inspect https://example.com/page --site https://example.com --language fr`,
	Args: cobra.ExactArgs(1),
	RunE: runInspect,
}

func init() {
	inspectCmd.Flags().StringVar(&inspectSite, "site", "", "Site URL the page belongs to (required)")
	inspectCmd.Flags().StringVar(&inspectLanguage, "language", "", "Language code for localized results (e.g. en, fr)")
	inspectCmd.MarkFlagRequired("site")
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(cmd *cobra.Command, args []string) error {
	url := args[0]

	req := &gsc.InspectUrlIndexRequest{
		InspectionUrl: url,
		SiteUrl:       inspectSite,
		LanguageCode:  inspectLanguage,
	}

	resp, err := svc.UrlInspection.Index.Inspect(req).Do()
	if err != nil {
		return fmt.Errorf("inspecting URL %q: %w", url, err)
	}

	if output.IsJSON(cmd) {
		return output.PrintJSON(resp, output.IsPretty(cmd))
	}

	result := resp.InspectionResult
	if result == nil {
		fmt.Println("No inspection result returned.")
		return nil
	}

	fmt.Printf("URL: %s\n\n", url)

	// Index status.
	if idx := result.IndexStatusResult; idx != nil {
		fmt.Println("=== Index Status ===")
		output.PrintKeyValue([][]string{
			{"Coverage State:", idx.CoverageState},
			{"Indexing State:", idx.IndexingState},
			{"Robots.txt State:", idx.RobotsTxtState},
			{"Page Fetch State:", idx.PageFetchState},
			{"Last Crawl Time:", idx.LastCrawlTime},
			{"Google Canonical:", idx.GoogleCanonical},
			{"User Canonical:", idx.UserCanonical},
			{"Crawled As:", idx.CrawledAs},
		})
		if len(idx.Sitemap) > 0 {
			fmt.Printf("Sitemaps: %v\n", idx.Sitemap)
		}
		if len(idx.ReferringUrls) > 0 {
			fmt.Printf("Referring URLs: %v\n", idx.ReferringUrls)
		}
		fmt.Println()
	}

	// Mobile usability.
	if mob := result.MobileUsabilityResult; mob != nil {
		fmt.Println("=== Mobile Usability ===")
		output.PrintKeyValue([][]string{
			{"Verdict:", mob.Verdict},
		})
		if len(mob.Issues) > 0 {
			fmt.Println("Issues:")
			for _, issue := range mob.Issues {
				fmt.Printf("  - [%s] %s\n", issue.IssueType, issue.Message)
			}
		}
		fmt.Println()
	}

	// Rich results.
	if rich := result.RichResultsResult; rich != nil {
		fmt.Println("=== Rich Results ===")
		output.PrintKeyValue([][]string{
			{"Verdict:", rich.Verdict},
		})
		if len(rich.DetectedItems) > 0 {
			fmt.Println("Detected items:")
			for _, item := range rich.DetectedItems {
				fmt.Printf("  - %s (%d items)\n", item.RichResultType, len(item.Items))
				for _, it := range item.Items {
					if it.Name != "" {
						fmt.Printf("      • %s\n", it.Name)
					}
					for _, iss := range it.Issues {
						fmt.Printf("        ! [%s] %s\n", iss.Severity, iss.IssueMessage)
					}
				}
			}
		}
	}

	return nil
}
