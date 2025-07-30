package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
	"github.com/spf13/cobra"
)

func newLogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show commit logs",
		Long:  `Shows the commit logs starting from the current HEAD.`,
		RunE:  runLog,
	}

	cmd.Flags().IntP("max-count", "n", 0, "Limit the number of commits to output")
	cmd.Flags().Bool("oneline", false, "Show each commit on a single line")
	cmd.Flags().Bool("graph", false, "Show a text-based graphical representation of the commit history")
	cmd.Flags().StringP("pretty", "", "", "Pretty-print the contents of the commit logs")

	return cmd
}

func runLog(cmd *cobra.Command, args []string) error {
	// Find repository
	repoPath, err := findRepository()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	// Open repository
	repo, err := vcs.Open(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Get flags
	maxCount, _ := cmd.Flags().GetInt("max-count")
	oneline, _ := cmd.Flags().GetBool("oneline")
	showGraph, _ := cmd.Flags().GetBool("graph")
	prettyFormat, _ := cmd.Flags().GetString("pretty")

	// Get reference manager
	refManager := refs.NewRefManager(repo.GitDir())

	// Get current HEAD
	currentCommitID, _, err := refManager.HEAD()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	if currentCommitID.IsZero() {
		fmt.Println("No commits found")
		return nil
	}

	// Walk commit history
	commitCount := 0
	commitID := currentCommitID

	for !commitID.IsZero() {
		if maxCount > 0 && commitCount >= maxCount {
			break
		}

		// Read commit object
		obj, err := repo.ReadObject(commitID)
		if err != nil {
			return fmt.Errorf("failed to read commit %s: %w", commitID.String(), err)
		}

		commit, ok := obj.(*objects.Commit)
		if !ok {
			return fmt.Errorf("object %s is not a commit", commitID.String())
		}

		// Print commit
		if oneline {
			printCommitOneline(commitID, commit)
		} else if prettyFormat != "" {
			printCommitPretty(commitID, commit, prettyFormat)
		} else {
			printCommitFull(commitID, commit, showGraph, commitCount == 0)
		}

		// Get parent commit
		parents := commit.Parents()
		if len(parents) == 0 {
			break
		}

		// For now, just follow the first parent
		commitID = parents[0]
		commitCount++
	}

	return nil
}

func printCommitOneline(commitID objects.ObjectID, commit *objects.Commit) {
	message := strings.Split(strings.TrimSpace(commit.Message()), "\n")[0]
	fmt.Printf("%s %s\n", commitID.String()[:7], message)
}

func printCommitFull(commitID objects.ObjectID, commit *objects.Commit, showGraph bool, isFirst bool) {
	prefix := ""
	if showGraph {
		if isFirst {
			prefix = "* "
		} else {
			prefix = "* "
		}
	}

	fmt.Printf("%scommit %s\n", prefix, commitID.String())

	parents := commit.Parents()
	if len(parents) > 1 {
		fmt.Printf("Merge:")
		for _, parent := range parents {
			fmt.Printf(" %s", parent.String()[:7])
		}
		fmt.Println()
	}

	fmt.Printf("Author: %s <%s>\n", commit.Author().Name, commit.Author().Email)
	fmt.Printf("Date:   %s\n", formatDate(commit.Author().When))
	fmt.Println()

	// Print commit message with indentation
	messageLines := strings.Split(strings.TrimSpace(commit.Message()), "\n")
	for _, line := range messageLines {
		fmt.Printf("    %s\n", line)
	}
	fmt.Println()
}

func printCommitPretty(commitID objects.ObjectID, commit *objects.Commit, format string) {
	// Simple pretty format implementation
	switch format {
	case "oneline":
		printCommitOneline(commitID, commit)
	case "short":
		fmt.Printf("commit %s\n", commitID.String()[:7])
		fmt.Printf("Author: %s\n", commit.Author().Name)
		fmt.Printf("\n    %s\n\n", strings.TrimSpace(commit.Message()))
	default:
		printCommitFull(commitID, commit, false, true)
	}
}

func formatDate(t time.Time) string {
	// Format like Git's default date format
	return t.Format("Mon Jan 2 15:04:05 2006 -0700")
}