package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func newTagCommand() *cobra.Command {
	var (
		list      bool
		delete    bool
		annotated bool
		message   string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "tag [flags] [<tagname>] [<commit>]",
		Short: "Create, list, delete or verify a tag object signed with GPG",
		Long: `Create, list, delete tags. Tags are refs that point to specific points in Git history.
Lightweight tags are simple references to commits, while annotated tags are objects with metadata.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			vcsRepo, err := vcs.Open(repo)
			if err != nil {
				return err
			}

			refManager := refs.NewRefManager(vcsRepo.GitDir())

			if list || len(args) == 0 {
				return listTags(vcsRepo, refManager)
			}

			tagName := args[0]

			if delete {
				return deleteTag(vcsRepo, refManager, tagName)
			}

			// Create tag
			target := "HEAD"
			if len(args) > 1 {
				target = args[1]
			}

			return createTag(vcsRepo, refManager, tagName, target, annotated, message, force)
		},
	}

	cmd.Flags().BoolVarP(&list, "list", "l", false, "List tags")
	cmd.Flags().BoolVarP(&delete, "delete", "d", false, "Delete tag")
	cmd.Flags().BoolVarP(&annotated, "annotate", "a", false, "Create annotated tag")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Tag message")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Replace existing tag")

	return cmd
}

func listTags(repo *vcs.Repository, refManager *refs.RefManager) error {
	tagsDir := filepath.Join(repo.GitDir(), "refs", "tags")
	
	// Check if tags directory exists
	if _, err := os.Stat(tagsDir); os.IsNotExist(err) {
		return nil // No tags
	}

	var tags []string
	err := filepath.Walk(tagsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(tagsDir, path)
			if err != nil {
				return err
			}
			tags = append(tags, relPath)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	sort.Strings(tags)
	for _, tag := range tags {
		fmt.Println(tag)
	}

	return nil
}

func createTag(repo *vcs.Repository, refManager *refs.RefManager, tagName, target string, annotated bool, message string, force bool) error {
	// Validate tag name
	if err := validateTagName(tagName); err != nil {
		return err
	}

	// Check if tag already exists
	tagRef := "refs/tags/" + tagName
	if refManager.RefExists(tagName) && !force {
		return fmt.Errorf("tag '%s' already exists", tagName)
	}

	// Resolve target commit
	targetID, err := refManager.ResolveRef(target)
	if err != nil {
		return fmt.Errorf("failed to resolve %q: %w", target, err)
	}

	// Verify target is a commit
	_, err = repo.GetCommit(targetID)
	if err != nil {
		return fmt.Errorf("target %s is not a commit: %w", target, err)
	}

	var tagObjectID objects.ObjectID

	if annotated || message != "" {
		// Create annotated tag
		if message == "" {
			message = fmt.Sprintf("Tag %s", tagName)
		}

		tagger := objects.Signature{
			Name:  "VCS User",
			Email: "user@example.com",
			When:  time.Now(),
		}

		tagObj, err := repo.CreateTag(targetID, objects.TypeCommit, tagName, tagger, message)
		if err != nil {
			return fmt.Errorf("failed to create tag object: %w", err)
		}

		tagObjectID = tagObj.ID()
		fmt.Printf("Created annotated tag %s\n", tagName)
	} else {
		// Create lightweight tag (just a ref)
		tagObjectID = targetID
		fmt.Printf("Created lightweight tag %s\n", tagName)
	}

	// Write tag reference
	if err := refManager.WriteRef(tagRef, tagObjectID, nil); err != nil {
		return fmt.Errorf("failed to write tag reference: %w", err)
	}

	return nil
}

func deleteTag(repo *vcs.Repository, refManager *refs.RefManager, tagName string) error {
	tagRef := "refs/tags/" + tagName
	
	if !refManager.RefExists(tagName) {
		return fmt.Errorf("tag '%s' not found", tagName)
	}

	tagPath := filepath.Join(repo.GitDir(), tagRef)
	if err := os.Remove(tagPath); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	fmt.Printf("Deleted tag '%s'\n", tagName)
	return nil
}

func validateTagName(name string) error {
	if name == "" {
		return fmt.Errorf("tag name cannot be empty")
	}

	// Basic validation - Git has more complex rules
	if strings.Contains(name, " ") {
		return fmt.Errorf("tag name cannot contain spaces")
	}

	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("tag name cannot start with '-'")
	}

	if strings.Contains(name, "..") {
		return fmt.Errorf("tag name cannot contain '..'")
	}

	// Prevent some problematic characters
	invalidChars := []string{"~", "^", ":", "?", "*", "[", "\\"}
	for _, invalid := range invalidChars {
		if strings.Contains(name, invalid) {
			return fmt.Errorf("tag name cannot contain '%s'", invalid)
		}
	}

	return nil
}