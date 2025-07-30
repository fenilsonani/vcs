package main

import (
	"fmt"
	"os"

	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/pkg/vcs"
	"github.com/spf13/cobra"
)

func newCatFileCommand() *cobra.Command {
	var (
		showType    bool
		showSize    bool
		showContent bool
		pretty      bool
	)
	
	cmd := &cobra.Command{
		Use:   "cat-file [options] <object>",
		Short: "Provide content or type and size information for repository objects",
		Long:  "Display the content, type, or size of repository objects",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Open repository
			repo, err := vcs.Open(".")
			if err != nil {
				return fmt.Errorf("not in a vcs repository: %w", err)
			}
			
			// Parse object ID
			id, err := objects.NewObjectID(args[0])
			if err != nil {
				return fmt.Errorf("invalid object ID: %w", err)
			}
			
			// Read object
			obj, err := repo.ReadObject(id)
			if err != nil {
				return fmt.Errorf("failed to read object: %w", err)
			}
			
			// Handle different output modes
			switch {
			case showType:
				fmt.Println(obj.Type())
			case showSize:
				fmt.Println(obj.Size())
			case showContent || pretty:
				data, err := obj.Serialize()
				if err != nil {
					return fmt.Errorf("failed to serialize object: %w", err)
				}
				os.Stdout.Write(data)
			default:
				return fmt.Errorf("must specify one of -t, -s, -e, or -p")
			}
			
			return nil
		},
	}
	
	cmd.Flags().BoolVarP(&showType, "type", "t", false, "Show object type")
	cmd.Flags().BoolVarP(&showSize, "size", "s", false, "Show object size")
	cmd.Flags().BoolVarP(&showContent, "exist", "e", false, "Exit with zero status if object exists")
	cmd.Flags().BoolVarP(&pretty, "pretty-print", "p", false, "Pretty-print object content")
	
	return cmd
}