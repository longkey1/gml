/*
Copyright © 2025 longkey1

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/longkey1/gml/internal/gml"
	"github.com/spf13/cobra"
)

// modifyCmd represents the modify command
var modifyCmd = &cobra.Command{
	Use:   "modify <message-id>",
	Short: "Modify a Gmail message's labels",
	Long: `Modify a Gmail message by adding or removing labels.

Supports marking messages as read/unread and archiving/unarchiving.

Examples:
  gml modify 18abc123def456 --read                        # Mark as read
  gml modify 18abc123def456 --unread                      # Mark as unread
  gml modify 18abc123def456 --archive                     # Archive message
  gml modify 18abc123def456 --unarchive                   # Move back to inbox
  gml modify 18abc123def456 --read --archive              # Read and archive
  gml modify 18abc123def456 --add-label MyLabel           # Add a custom label
  gml modify 18abc123def456 --remove-label MyLabel        # Remove a custom label
  gml modify 18abc123def456 --add-label Label1,Label2     # Add multiple labels`,
	Annotations: map[string]string{"write": "true"},
	Args:        cobra.ExactArgs(1),
	RunE:        runModify,
}

func runModify(cmd *cobra.Command, args []string) error {
	messageID := args[0]
	ctx := cmd.Context()
	cfg := GetConfig()

	// Get flags
	read, _ := cmd.Flags().GetBool("read")
	unread, _ := cmd.Flags().GetBool("unread")
	archive, _ := cmd.Flags().GetBool("archive")
	unarchive, _ := cmd.Flags().GetBool("unarchive")
	addLabels, _ := cmd.Flags().GetStringSlice("add-label")
	removeLabels, _ := cmd.Flags().GetStringSlice("remove-label")

	// Validate conflicting flags
	if read && unread {
		return fmt.Errorf("cannot specify both --read and --unread")
	}
	if archive && unarchive {
		return fmt.Errorf("cannot specify both --archive and --unarchive")
	}

	// Validate at least one flag is specified
	if !read && !unread && !archive && !unarchive && len(addLabels) == 0 && len(removeLabels) == 0 {
		return fmt.Errorf("at least one of --read, --unread, --archive, --unarchive, --add-label, --remove-label must be specified")
	}

	// Build label modifications
	var addLabelIDs, removeLabelIDs []string

	if read {
		removeLabelIDs = append(removeLabelIDs, "UNREAD")
	}
	if unread {
		addLabelIDs = append(addLabelIDs, "UNREAD")
	}
	if archive {
		removeLabelIDs = append(removeLabelIDs, "INBOX")
	}
	if unarchive {
		addLabelIDs = append(addLabelIDs, "INBOX")
	}

	// Create service
	svc, err := gml.NewService(ctx, cfg)
	if err != nil {
		return fmt.Errorf("unable to create service: %w", err)
	}

	// Resolve custom label names to IDs if needed
	if len(addLabels) > 0 || len(removeLabels) > 0 {
		labelIndex, err := gml.FetchLabelIndex(svc)
		if err != nil {
			return fmt.Errorf("unable to fetch labels: %w", err)
		}
		if len(addLabels) > 0 {
			ids, err := labelIndex.ResolveLabelIDs(addLabels)
			if err != nil {
				return fmt.Errorf("unable to resolve add labels: %w", err)
			}
			addLabelIDs = append(addLabelIDs, ids...)
		}
		if len(removeLabels) > 0 {
			ids, err := labelIndex.ResolveLabelIDs(removeLabels)
			if err != nil {
				return fmt.Errorf("unable to resolve remove labels: %w", err)
			}
			removeLabelIDs = append(removeLabelIDs, ids...)
		}
	}

	// Modify message
	if err := gml.ModifyMessage(ctx, svc, messageID, addLabelIDs, removeLabelIDs); err != nil {
		return fmt.Errorf("unable to modify message: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Message modified successfully.")
	return nil
}

func init() {
	rootCmd.AddCommand(modifyCmd)

	modifyCmd.Flags().Bool("read", false, "Mark message as read (removes UNREAD label)")
	modifyCmd.Flags().Bool("unread", false, "Mark message as unread (adds UNREAD label)")
	modifyCmd.Flags().Bool("archive", false, "Archive message (removes INBOX label)")
	modifyCmd.Flags().Bool("unarchive", false, "Unarchive message (adds INBOX label)")
	modifyCmd.Flags().StringSlice("add-label", nil, "Add label to message (can be specified multiple times)")
	modifyCmd.Flags().StringSlice("remove-label", nil, "Remove label from message (can be specified multiple times)")

	// Set custom output to enable testing
	modifyCmd.SetOut(os.Stdout)
}
