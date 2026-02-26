package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the20100/g-search-console-cli/internal/output"
)

var sitemapsCmd = &cobra.Command{
	Use:   "sitemaps",
	Short: "Manage sitemaps for a site",
}

var sitemapsListCmd = &cobra.Command{
	Use:   "list <site-url>",
	Short: "List all sitemaps for a site",
	Args:  cobra.ExactArgs(1),
	RunE:  runSitemapsList,
}

var sitemapsGetCmd = &cobra.Command{
	Use:   "get <site-url> <feedpath>",
	Short: "Get details for a specific sitemap",
	Args:  cobra.ExactArgs(2),
	RunE:  runSitemapsGet,
}

var sitemapsSubmitCmd = &cobra.Command{
	Use:   "submit <site-url> <feedpath>",
	Short: "Submit a sitemap for a site",
	Example: "  gsc sitemaps submit https://example.com https://example.com/sitemap.xml",
	Args:  cobra.ExactArgs(2),
	RunE:  runSitemapsSubmit,
}

var sitemapsDeleteCmd = &cobra.Command{
	Use:   "delete <site-url> <feedpath>",
	Short: "Delete a sitemap for a site",
	Args:  cobra.ExactArgs(2),
	RunE:  runSitemapsDelete,
}

func init() {
	sitemapsCmd.AddCommand(sitemapsListCmd, sitemapsGetCmd, sitemapsSubmitCmd, sitemapsDeleteCmd)
	rootCmd.AddCommand(sitemapsCmd)
}

func runSitemapsList(cmd *cobra.Command, args []string) error {
	siteURL := args[0]
	resp, err := svc.Sitemaps.List(siteURL).Do()
	if err != nil {
		return fmt.Errorf("listing sitemaps for %q: %w", siteURL, err)
	}

	if output.IsJSON(cmd) {
		return output.PrintJSON(resp.Sitemap, output.IsPretty(cmd))
	}

	if len(resp.Sitemap) == 0 {
		fmt.Println("No sitemaps found.")
		return nil
	}

	headers := []string{"PATH", "TYPE", "LAST SUBMITTED", "LAST DOWNLOADED", "WARNINGS", "ERRORS"}
	rows := make([][]string, len(resp.Sitemap))
	for i, s := range resp.Sitemap {
		rows[i] = []string{
			s.Path,
			s.Type,
			s.LastSubmitted,
			s.LastDownloaded,
			fmt.Sprintf("%d", s.Warnings),
			fmt.Sprintf("%d", s.Errors),
		}
	}
	output.PrintTable(headers, rows)
	return nil
}

func runSitemapsGet(cmd *cobra.Command, args []string) error {
	siteURL := args[0]
	feedpath := args[1]

	sitemap, err := svc.Sitemaps.Get(siteURL, feedpath).Do()
	if err != nil {
		return fmt.Errorf("getting sitemap %q for %q: %w", feedpath, siteURL, err)
	}

	if output.IsJSON(cmd) {
		return output.PrintJSON(sitemap, output.IsPretty(cmd))
	}

	contentSummary := make([]string, len(sitemap.Contents))
	for i, c := range sitemap.Contents {
		contentSummary[i] = fmt.Sprintf("%s (%d submitted, %d indexed)", c.Type, c.Submitted, c.Indexed)
	}

	output.PrintKeyValue([][]string{
		{"Path:", sitemap.Path},
		{"Type:", sitemap.Type},
		{"Last Submitted:", sitemap.LastSubmitted},
		{"Last Downloaded:", sitemap.LastDownloaded},
		{"Is Pending:", fmt.Sprintf("%v", sitemap.IsPending)},
		{"Is Sitemaps Index:", fmt.Sprintf("%v", sitemap.IsSitemapsIndex)},
		{"Errors:", fmt.Sprintf("%d", sitemap.Errors)},
		{"Warnings:", fmt.Sprintf("%d", sitemap.Warnings)},
		{"Contents:", strings.Join(contentSummary, "; ")},
	})
	return nil
}

func runSitemapsSubmit(cmd *cobra.Command, args []string) error {
	siteURL := args[0]
	feedpath := args[1]

	if err := svc.Sitemaps.Submit(siteURL, feedpath).Do(); err != nil {
		return fmt.Errorf("submitting sitemap %q for %q: %w", feedpath, siteURL, err)
	}
	fmt.Printf("Sitemap submitted: %s\n", feedpath)
	return nil
}

func runSitemapsDelete(cmd *cobra.Command, args []string) error {
	siteURL := args[0]
	feedpath := args[1]

	if err := svc.Sitemaps.Delete(siteURL, feedpath).Do(); err != nil {
		return fmt.Errorf("deleting sitemap %q for %q: %w", feedpath, siteURL, err)
	}
	fmt.Printf("Sitemap deleted: %s\n", feedpath)
	return nil
}
