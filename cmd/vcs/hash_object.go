package main

import (
	"fmt"
	"io"
	"os"

	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/pkg/vcs"
	"github.com/spf13/cobra"
)

func newHashObjectCommand() *cobra.Command {
	var (
		write  bool
		stdin  bool
		objType string
	)
	
	cmd := &cobra.Command{
		Use:   "hash-object [file...]",
		Short: "Compute object ID and optionally creates a blob from a file",
		Long:  "Computes the object ID value for an object with specified type and optionally writes it to the object database",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate object type
			if objType != "blob" {
				return fmt.Errorf("only blob type is currently supported")
			}
			
			// Open repository if writing
			var repo *vcs.Repository
			if write {
				var err error
				repo, err = vcs.Open(".")
				if err != nil {
					return fmt.Errorf("not in a vcs repository: %w", err)
				}
			}
			
			// Process stdin or files
			if stdin || len(args) == 0 {
				id, err := hashObject(repo, os.Stdin, objects.TypeBlob, write)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), id)
			} else {
				// Process each file
				for _, path := range args {
					file, err := os.Open(path)
					if err != nil {
						return fmt.Errorf("failed to open %s: %w", path, err)
					}
					
					id, err := hashObject(repo, file, objects.TypeBlob, write)
					file.Close()
					
					if err != nil {
						return fmt.Errorf("failed to hash %s: %w", path, err)
					}
					
					fmt.Fprintln(cmd.OutOrStdout(), id)
				}
			}
			
			return nil
		},
	}
	
	cmd.Flags().BoolVarP(&write, "write", "w", false, "Actually write the object into the object database")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read from stdin instead of from a file")
	cmd.Flags().StringVarP(&objType, "type", "t", "blob", "Specify the type of object to be created")
	
	return cmd
}

func hashObject(repo *vcs.Repository, reader io.Reader, objType objects.ObjectType, write bool) (objects.ObjectID, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return objects.ObjectID{}, fmt.Errorf("failed to read data: %w", err)
	}
	
	if repo != nil && write {
		return repo.HashObject(data, objType, true)
	}
	
	// Just compute hash without writing
	obj := objects.NewBlob(data)
	return obj.ID(), nil
}