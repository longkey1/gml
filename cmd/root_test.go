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
	"testing"

	"github.com/longkey1/gml/internal/gml"
	"github.com/spf13/cobra"
)

func TestPersistentPreRunE_ReadOnlyGuard(t *testing.T) {
	writeCmd := &cobra.Command{Annotations: map[string]string{"write": "true"}}
	readCmd := &cobra.Command{}

	tests := []struct {
		name    string
		config  *gml.Config
		cmd     *cobra.Command
		wantErr bool
	}{
		{
			name:    "read-only blocks write command",
			config:  &gml.Config{ReadOnly: true},
			cmd:     writeCmd,
			wantErr: true,
		},
		{
			name:    "read-only allows non-write command",
			config:  &gml.Config{ReadOnly: true},
			cmd:     readCmd,
			wantErr: false,
		},
		{
			name:    "non-read-only allows write command",
			config:  &gml.Config{ReadOnly: false},
			cmd:     writeCmd,
			wantErr: false,
		},
		{
			name:    "nil config allows write command",
			config:  nil,
			cmd:     writeCmd,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config = tt.config
			t.Cleanup(func() { config = nil })

			err := rootCmd.PersistentPreRunE(tt.cmd, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("PersistentPreRunE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
