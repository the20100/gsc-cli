package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/the20100/g-search-console-cli/internal/output"
	gsc "google.golang.org/api/searchconsole/v1"
)

var mobileTestCmd = &cobra.Command{
	Use:   "mobile-test <url>",
	Short: "Run a mobile-friendly test for a URL",
	Long: `Test whether a URL is mobile-friendly according to Google's criteria.

Example:
  gsc mobile-test https://example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runMobileTest,
}

func init() {
	rootCmd.AddCommand(mobileTestCmd)
}

func runMobileTest(cmd *cobra.Command, args []string) error {
	url := args[0]

	req := &gsc.RunMobileFriendlyTestRequest{
		Url: url,
	}

	resp, err := svc.UrlTestingTools.MobileFriendlyTest.Run(req).Do()
	if err != nil {
		return fmt.Errorf("running mobile-friendly test for %q: %w", url, err)
	}

	if output.IsJSON(cmd) {
		return output.PrintJSON(resp, output.IsPretty(cmd))
	}

	fmt.Printf("URL: %s\n\n", url)

	testStatus := ""
	if resp.TestStatus != nil {
		testStatus = resp.TestStatus.Status
		if resp.TestStatus.Details != "" {
			testStatus += " — " + resp.TestStatus.Details
		}
	}

	output.PrintKeyValue([][]string{
		{"Test Status:", testStatus},
		{"Mobile Friendliness:", resp.MobileFriendliness},
	})

	if len(resp.MobileFriendlyIssues) > 0 {
		fmt.Println("\nMobile-Friendly Issues:")
		for _, issue := range resp.MobileFriendlyIssues {
			fmt.Printf("  - %s\n", issue.Rule)
		}
	}

	if len(resp.ResourceIssues) > 0 {
		fmt.Println("\nResource Issues (blocked resources):")
		for _, ri := range resp.ResourceIssues {
			if ri.BlockedResource != nil {
				fmt.Printf("  - blocked: %s\n", ri.BlockedResource.Url)
			}
		}
	}

	return nil
}
