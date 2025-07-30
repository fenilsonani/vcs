#!/bin/bash

# Script to fix output capture issues in CLI commands
# Replace fmt.Printf with fmt.Fprintf(cmd.OutOrStdout(), ...)

find /Users/fenilsonani/Developer/fenilcom/projects/vcs/cmd/vcs -name "*.go" -exec sed -i '' 's/\bfmt\.Printf(/fmt.Fprintf(cmd.OutOrStdout(), /g' {} \;

echo "Fixed all fmt.Printf calls in CLI commands"