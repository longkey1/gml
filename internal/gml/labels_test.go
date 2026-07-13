package gml

import (
	"reflect"
	"strings"
	"testing"
)

// newLabelIndex builds a LabelIndex from name -> ID pairs the same
// way FetchLabelIndex does, without calling the Gmail API.
func newLabelIndex(labels map[string]string) *LabelIndex {
	nameToID := make(map[string]string)
	idToName := make(map[string]string)
	idToID := make(map[string]string)
	for name, id := range labels {
		nameToID[strings.ToLower(name)] = id
		idToName[strings.ToLower(id)] = name
		idToID[strings.ToLower(id)] = id
	}
	return &LabelIndex{
		nameToID: nameToID,
		idToName: idToName,
		idToID:   idToID,
	}
}

func TestResolveLabelIDs(t *testing.T) {
	t.Parallel()

	idx := newLabelIndex(map[string]string{
		"INBOX":     "INBOX",
		"SENT":      "SENT",
		"Work/Todo": "Label_123",
	})

	tests := []struct {
		name      string
		idx       *LabelIndex
		requested []string
		want      []string
		wantErr   string
	}{
		{
			name:      "system label by name",
			idx:       idx,
			requested: []string{"INBOX"},
			want:      []string{"INBOX"},
		},
		{
			name:      "case insensitive name",
			idx:       idx,
			requested: []string{"inbox", "Sent"},
			want:      []string{"INBOX", "SENT"},
		},
		{
			name:      "custom label by name",
			idx:       idx,
			requested: []string{"work/todo"},
			want:      []string{"Label_123"},
		},
		{
			name:      "custom label by ID",
			idx:       idx,
			requested: []string{"label_123"},
			want:      []string{"Label_123"},
		},
		{
			name:      "surrounding whitespace is trimmed",
			idx:       idx,
			requested: []string{"  INBOX  "},
			want:      []string{"INBOX"},
		},
		{
			name:      "empty input",
			idx:       idx,
			requested: nil,
			want:      nil,
		},
		{
			name:      "unknown label",
			idx:       idx,
			requested: []string{"NoSuchLabel"},
			wantErr:   "label not found: NoSuchLabel",
		},
		{
			name:      "unknown label among known ones",
			idx:       idx,
			requested: []string{"INBOX", "NoSuchLabel"},
			wantErr:   "label not found: NoSuchLabel",
		},
		{
			name:      "nil index",
			idx:       nil,
			requested: []string{"INBOX"},
			wantErr:   "label index is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.idx.ResolveLabelIDs(tt.requested)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ResolveLabelIDs() error = nil, want error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ResolveLabelIDs() error = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveLabelIDs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveLabelIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapLabelIDsToNames(t *testing.T) {
	t.Parallel()

	idx := newLabelIndex(map[string]string{
		"INBOX":     "INBOX",
		"Work/Todo": "Label_123",
	})

	tests := []struct {
		name string
		idx  *LabelIndex
		ids  []string
		want []string
	}{
		{
			name: "known IDs",
			idx:  idx,
			ids:  []string{"INBOX", "Label_123"},
			want: []string{"INBOX", "Work/Todo"},
		},
		{
			name: "case insensitive ID lookup",
			idx:  idx,
			ids:  []string{"label_123"},
			want: []string{"Work/Todo"},
		},
		{
			name: "unknown ID falls back to the ID",
			idx:  idx,
			ids:  []string{"Label_999"},
			want: []string{"Label_999"},
		},
		{
			name: "mixed known and unknown",
			idx:  idx,
			ids:  []string{"INBOX", "Label_999"},
			want: []string{"INBOX", "Label_999"},
		},
		{
			name: "empty input",
			idx:  idx,
			ids:  nil,
			want: nil,
		},
		{
			name: "nil index returns IDs as-is",
			idx:  nil,
			ids:  []string{"INBOX", "Label_123"},
			want: []string{"INBOX", "Label_123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.idx.MapLabelIDsToNames(tt.ids)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapLabelIDsToNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildMailURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		email    string
		threadID string
		want     string
	}{
		{
			name:     "typical values",
			email:    "user@example.com",
			threadID: "thread123",
			want:     "https://mail.google.com/mail/?authuser=user@example.com#all/thread123",
		},
		{
			name:     "empty values",
			email:    "",
			threadID: "",
			want:     "https://mail.google.com/mail/?authuser=#all/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := BuildMailURL(tt.email, tt.threadID); got != tt.want {
				t.Errorf("BuildMailURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
