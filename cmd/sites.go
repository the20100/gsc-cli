package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/the20100/g-search-console-cli/internal/output"
)

var sitesCmd = &cobra.Command{
	Use:   "sites",
	Short: "Manage Search Console site properties",
}

var sitesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all verified site properties",
	RunE:  runSitesList,
}

var sitesGetCmd = &cobra.Command{
	Use:   "get <site-url>",
	Short: "Get details for a specific site property",
	Args:  cobra.ExactArgs(1),
	RunE:  runSitesGet,
}

var sitesAddCmd = &cobra.Command{
	Use:   "add <site-url>",
	Short: "Add a site property to Search Console",
	Args:  cobra.ExactArgs(1),
	RunE:  runSitesAdd,
}

var sitesDeleteCmd = &cobra.Command{
	Use:   "delete <site-url>",
	Short: "Delete a site property from Search Console",
	Args:  cobra.ExactArgs(1),
	RunE:  runSitesDelete,
}

func init() {
	sitesCmd.AddCommand(sitesListCmd, sitesGetCmd, sitesAddCmd, sitesDeleteCmd)
	rootCmd.AddCommand(sitesCmd)
}

func runSitesList(cmd *cobra.Command, args []string) error {
	resp, err := svc.Sites.List().Do()
	if err != nil {
		return fmt.Errorf("listing sites: %w", err)
	}

	if output.IsJSON(cmd) {
		return output.PrintJSON(resp.SiteEntry, output.IsPretty(cmd))
	}

	if len(resp.SiteEntry) == 0 {
		fmt.Println("No sites found.")
		return nil
	}

	headers := []string{"SITE URL", "PERMISSION LEVEL"}
	rows := make([][]string, len(resp.SiteEntry))
	for i, s := range resp.SiteEntry {
		rows[i] = []string{s.SiteUrl, s.PermissionLevel}
	}
	output.PrintTable(headers, rows)
	return nil
}

func runSitesGet(cmd *cobra.Command, args []string) error {
	siteURL := args[0]
	site, err := svc.Sites.Get(siteURL).Do()
	if err != nil {
		return fmt.Errorf("getting site %q: %w", siteURL, err)
	}

	if output.IsJSON(cmd) {
		return output.PrintJSON(site, output.IsPretty(cmd))
	}

	output.PrintKeyValue([][]string{
		{"Site URL:", site.SiteUrl},
		{"Permission Level:", site.PermissionLevel},
	})
	return nil
}

func runSitesAdd(cmd *cobra.Command, args []string) error {
	siteURL := args[0]
	if err := svc.Sites.Add(siteURL).Do(); err != nil {
		return fmt.Errorf("adding site %q: %w", siteURL, err)
	}
	fmt.Printf("Site added: %s\n", siteURL)
	return nil
}

func runSitesDelete(cmd *cobra.Command, args []string) error {
	siteURL := args[0]
	if err := svc.Sites.Delete(siteURL).Do(); err != nil {
		return fmt.Errorf("deleting site %q: %w", siteURL, err)
	}
	fmt.Printf("Site deleted: %s\n", siteURL)
	return nil
}
