package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type DailyStats struct {
	FilesChanged map[string]struct{}
	Additions    int
	Deletions    int
}

const (
	colorReset  = "\033[0m"
	colorOrange = "\033[38;5;208m"
	colorCyan   = "\033[36m"
)

const (
	dateRangeWidth    = 25
	filesChangedWidth = 15
	additionsWidth    = 11
	deletionsWidth    = 11
	totalChangesWidth = 15
)

func getGitStats(repoPath string, startDate, endDate time.Time) (map[string]*DailyStats, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	endDate = endDate.Add(24 * time.Hour).Add(-time.Second)

	commits, err := repo.Log(&git.LogOptions{
		Since: &startDate,
		Until: &endDate,
	})
	if err != nil {
		return nil, err
	}

	dailyStats := make(map[string]*DailyStats)

	err = commits.ForEach(func(c *object.Commit) error {
		commitDate := c.Author.When.Format("2006-01-02")
		stats, err := c.Stats()
		if err != nil {
			return err
		}

		if _, ok := dailyStats[commitDate]; !ok {
			dailyStats[commitDate] = &DailyStats{
				FilesChanged: make(map[string]struct{}),
			}
		}

		for _, stat := range stats {
			dailyStats[commitDate].FilesChanged[stat.Name] = struct{}{}
			dailyStats[commitDate].Additions += stat.Addition
			dailyStats[commitDate].Deletions += stat.Deletion
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return dailyStats, nil
}

func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

func formatDateRange(startDate, endDate time.Time) string {
	startStr := startDate.Format("2006-01-02")
	if startDate.Year() == endDate.Year() {
		endStr := endDate.Format("01-02")
		return fmt.Sprintf("%s ~ %s", startStr, endStr)
	}
	endStr := endDate.Format("2006-01-02")
	return fmt.Sprintf("%s ~ %s", startStr, endStr)
}

func printTableHeader() {
	totalWidth := dateRangeWidth + filesChangedWidth + additionsWidth + deletionsWidth + totalChangesWidth + 4 // +4 for separators

	fmt.Printf("%s|%s|%s|%s|%s\n",
		centerText("Date Range", dateRangeWidth),
		centerText("Files Changed", filesChangedWidth),
		centerText("Additions", additionsWidth),
		centerText("Deletions", deletionsWidth),
		centerText("Total Changes", totalChangesWidth))

	fmt.Printf("%s\n", strings.Repeat("-", totalWidth))
}

func printTableRow(dateRange string, filesChanged, additions, deletions, totalChanges int) {
	fmt.Printf("%s|%s|%s|%s|%s\n",
		padText(dateRange, dateRangeWidth),
		centerText(fmt.Sprintf("%d", filesChanged), filesChangedWidth),
		centerText(fmt.Sprintf("%d", additions), additionsWidth),
		centerText(fmt.Sprintf("%d", deletions), deletionsWidth),
		centerText(fmt.Sprintf("%d", totalChanges), totalChangesWidth))

	totalWidth := dateRangeWidth + filesChangedWidth + additionsWidth + deletionsWidth + totalChangesWidth + 4
	fmt.Printf("%s\n", strings.Repeat("-", totalWidth))
}

func printNoChangeRow(dateRange string, days int) {
	var days_tip = "day"
	if days > 1 {
		days_tip = "days"
	}

	message := fmt.Sprintf("%d %s no commits", days, days_tip)

	totalWidth := dateRangeWidth + filesChangedWidth + additionsWidth + deletionsWidth + totalChangesWidth + 4 // +4 for separators

	fmt.Printf("%s%s%s\n", colorOrange, strings.Repeat("-", totalWidth), colorReset)

	fmt.Printf("%s%s%s|%s%s%s\n",
		colorOrange,
		padText(dateRange, dateRangeWidth),
		colorReset,
		colorOrange,
		centerText(message, totalWidth-dateRangeWidth-1),
		colorReset)

	fmt.Printf("%s\n", strings.Repeat("-", totalWidth))
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	leftPad := (width - len(text)) / 2
	rightPad := width - len(text) - leftPad
	return fmt.Sprintf("%s%s%s", strings.Repeat(" ", leftPad), text, strings.Repeat(" ", rightPad))
}

func padText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	return fmt.Sprintf("%s%s", text, strings.Repeat(" ", width-len(text)))
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: git-stat <repo_path> <start_date> <end_date>")
		fmt.Println("Example: git-stat /path/to/repo 2023-08-30 2023-09-01")
		os.Exit(1)
	}

	repoPath := os.Args[1]
	startDateStr := os.Args[2]
	endDateStr := os.Args[3]

	startDate, err := parseDate(startDateStr)
	if err != nil {
		fmt.Printf("Invalid start date format: %v\n", err)
		os.Exit(1)
	}

	endDate, err := parseDate(endDateStr)
	if err != nil {
		fmt.Printf("Invalid end date format: %v\n", err)
		os.Exit(1)
	}

	if endDate.Before(startDate) {
		fmt.Println("End date must be after start date")
		os.Exit(1)
	}

	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		fmt.Printf("Error resolving repository path: %v\n", err)
		os.Exit(1)
	}

	dailyStats, err := getGitStats(absPath, startDate, endDate)
	if err != nil {
		fmt.Printf("Error getting Git statistics: %v\n", err)
		os.Exit(1)
	}

	printTableHeader()

	var noChangeStartDate time.Time
	var noChangeDays int

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		stats, ok := dailyStats[dateStr]

		if !ok {
			if noChangeDays == 0 {
				noChangeStartDate = d
			}
			noChangeDays++
		} else {
			if noChangeDays > 0 {
				printNoChangeRow(formatDateRange(noChangeStartDate, d.AddDate(0, 0, -1)), noChangeDays)
				noChangeDays = 0
			}
			totalChanges := stats.Additions + stats.Deletions
			printTableRow(dateStr, len(stats.FilesChanged), stats.Additions, stats.Deletions, totalChanges)
		}
	}

	if noChangeDays > 0 {
		printNoChangeRow(formatDateRange(noChangeStartDate, endDate), noChangeDays)
	}
}
