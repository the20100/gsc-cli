package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/the20100/g-search-console-cli/internal/output"
	gsc "google.golang.org/api/searchconsole/v1"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Query search analytics data",
}

var (
	analyticsSite       string
	analyticsStartDate  string
	analyticsEndDate    string
	analyticsDimensions string
	analyticsSearchType string
	analyticsDataState  string
	analyticsLimit      int64
	analyticsStartRow   int64
)

var analyticsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query search analytics data for a site",
	Long: `Query search performance data (clicks, impressions, CTR, position).

Dimensions: date, query, page, country, device, searchAppearance, hour
Search types: web (default), image, video, news, discover, googleNews
Data states: final (default), all, hourlyAll

Examples:
  gsc analytics query --site https://example.com --start 2024-01-01 --end 2024-01-31
  gsc analytics query --site https://example.com --dimensions query --limit 100
  gsc analytics query --site https://example.com --dimensions query,page,country --search-type image
  gsc analytics query --site https://example.com --dimensions date --start 2024-01-01 --end 2024-03-31`,
	RunE: runAnalyticsQuery,
}

func init() {
	analyticsQueryCmd.Flags().StringVar(&analyticsSite, "site", "", "Site URL (required)")
	analyticsQueryCmd.Flags().StringVar(&analyticsStartDate, "start", "", "Start date YYYY-MM-DD (required)")
	analyticsQueryCmd.Flags().StringVar(&analyticsEndDate, "end", "", "End date YYYY-MM-DD (required)")
	analyticsQueryCmd.Flags().StringVar(&analyticsDimensions, "dimensions", "", "Comma-separated dimensions: date,query,page,country,device,searchAppearance,hour")
	analyticsQueryCmd.Flags().StringVar(&analyticsSearchType, "search-type", "web", "Search type: web, image, video, news, discover, googleNews")
	analyticsQueryCmd.Flags().StringVar(&analyticsDataState, "data-state", "", "Data state: final, all, hourlyAll")
	analyticsQueryCmd.Flags().Int64Var(&analyticsLimit, "limit", 1000, "Max rows to return (1–25000)")
	analyticsQueryCmd.Flags().Int64Var(&analyticsStartRow, "start-row", 0, "Start row offset for pagination")

	analyticsQueryCmd.MarkFlagRequired("site")
	analyticsQueryCmd.MarkFlagRequired("start")
	analyticsQueryCmd.MarkFlagRequired("end")

	analyticsCmd.AddCommand(analyticsQueryCmd)
	rootCmd.AddCommand(analyticsCmd)
}

func runAnalyticsQuery(cmd *cobra.Command, args []string) error {
	req := &gsc.SearchAnalyticsQueryRequest{
		StartDate:  analyticsStartDate,
		EndDate:    analyticsEndDate,
		RowLimit:   analyticsLimit,
		StartRow:   analyticsStartRow,
		SearchType: analyticsSearchType,
	}

	var dims []string
	if analyticsDimensions != "" {
		parts := strings.Split(analyticsDimensions, ",")
		for _, p := range parts {
			d := strings.TrimSpace(p)
			if d != "" {
				dims = append(dims, strings.ToUpper(d))
			}
		}
		req.Dimensions = dims
	}

	if analyticsDataState != "" {
		req.DataState = strings.ToUpper(analyticsDataState)
	}

	resp, err := svc.Searchanalytics.Query(analyticsSite, req).Do()
	if err != nil {
		return fmt.Errorf("querying search analytics: %w", err)
	}

	if output.IsJSON(cmd) {
		return output.PrintJSON(resp, output.IsPretty(cmd))
	}

	if len(resp.Rows) == 0 {
		fmt.Println("No data found for the specified parameters.")
		return nil
	}

	// Build table headers dynamically based on requested dimensions.
	headers := make([]string, 0, len(dims)+4)
	for _, d := range dims {
		headers = append(headers, d)
	}
	headers = append(headers, "CLICKS", "IMPRESSIONS", "CTR", "POSITION")

	rows := make([][]string, 0, len(resp.Rows))
	for _, row := range resp.Rows {
		r := make([]string, 0, len(dims)+4)
		r = append(r, row.Keys...)
		r = append(r,
			fmt.Sprintf("%.0f", row.Clicks),
			fmt.Sprintf("%.0f", row.Impressions),
			fmt.Sprintf("%.2f%%", row.Ctr*100),
			fmt.Sprintf("%.1f", row.Position),
		)
		rows = append(rows, r)
	}

	output.PrintTable(headers, rows)
	fmt.Printf("\n%d rows  |  aggregation: %s\n", len(resp.Rows), resp.ResponseAggregationType)
	return nil
}
