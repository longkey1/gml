package gml

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestFormatMessageListJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []MessageInfo
		want     string
	}{
		{
			name: "messages with omitempty fields",
			messages: []MessageInfo{
				{ID: "msg1", Subject: "hello", Labels: []string{"INBOX"}},
				{ID: "msg2"},
			},
			want: `[
  {
    "id": "msg1",
    "subject": "hello",
    "labels": [
      "INBOX"
    ]
  },
  {
    "id": "msg2"
  }
]
`,
		},
		{
			name:     "nil messages",
			messages: nil,
			want:     "null\n",
		},
		{
			name:     "empty slice",
			messages: []MessageInfo{},
			want:     "[]\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			if err := FormatMessageList(&buf, tt.messages, nil, OutputFormatJSON); err != nil {
				t.Fatalf("FormatMessageList() error = %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("FormatMessageList() output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMessageListTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messages    []MessageInfo
		fields      map[string]bool
		wantContain []string
		wantAbsent  []string
	}{
		{
			name: "selected columns only",
			messages: []MessageInfo{
				{ID: "msg1", Subject: "hello", From: "alice@example.com"},
			},
			fields:      ParseFields("id,subject"),
			wantContain: []string{"ID", "SUBJECT", "msg1", "hello"},
			wantAbsent:  []string{"FROM", "alice@example.com"},
		},
		{
			name: "labels are comma joined",
			messages: []MessageInfo{
				{ID: "msg1", Labels: []string{"INBOX", "Work/Todo"}},
			},
			fields:      ParseFields("id,labels"),
			wantContain: []string{"LABELS", "INBOX, Work/Todo"},
		},
		{
			name: "long subject is truncated with ellipsis",
			messages: []MessageInfo{
				{ID: "msg1", Subject: strings.Repeat("s", 50)},
			},
			fields:      ParseFields("id,subject"),
			wantContain: []string{strings.Repeat("s", 37) + "..."},
			wantAbsent:  []string{strings.Repeat("s", 38)},
		},
		{
			name: "body is printed after the table",
			messages: []MessageInfo{
				{ID: "msg1", Body: "the message body"},
			},
			fields:      ParseFields("id,body"),
			wantContain: []string{"=== msg1 ===", "the message body"},
		},
		{
			name: "empty body is skipped",
			messages: []MessageInfo{
				{ID: "msg1"},
			},
			fields:     ParseFields("id,body"),
			wantAbsent: []string{"==="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			if err := FormatMessageList(&buf, tt.messages, tt.fields, OutputFormatText); err != nil {
				t.Fatalf("FormatMessageList() error = %v", err)
			}
			got := buf.String()
			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("FormatMessageList() output = %q, want it to contain %q", got, want)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("FormatMessageList() output = %q, must not contain %q", got, absent)
				}
			}
		})
	}
}

func TestFormatMessageDetailJSON(t *testing.T) {
	t.Parallel()

	detail := &MessageDetail{
		ID:       "msg1",
		ThreadID: "thread1",
		URL:      "https://mail.google.com/mail/?authuser=user@example.com#all/thread1",
		From:     "alice@example.com",
		To:       "bob@example.com",
		Subject:  "hello",
		Date:     "Mon, 1 Jan 2026 00:00:00 +0900",
		Labels:   []string{"INBOX"},
		Body:     "the body",
	}

	var buf bytes.Buffer
	if err := FormatMessageDetail(&buf, detail, OutputFormatJSON); err != nil {
		t.Fatalf("FormatMessageDetail() error = %v", err)
	}

	var got MessageDetail
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if !reflect.DeepEqual(got, *detail) {
		t.Errorf("FormatMessageDetail() round-trip = %+v, want %+v", got, *detail)
	}
}

func TestFormatMessageDetailText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		detail *MessageDetail
		want   string
	}{
		{
			name: "all fields",
			detail: &MessageDetail{
				ID:       "msg1",
				ThreadID: "thread1",
				URL:      "https://example.com/mail",
				From:     "alice@example.com",
				To:       "bob@example.com",
				Subject:  "hello",
				Date:     "Mon, 1 Jan 2026 00:00:00 +0900",
				Labels:   []string{"INBOX", "Work/Todo"},
				Body:     "the body",
			},
			want: `ID: msg1
ThreadID: thread1
URL: https://example.com/mail
From: alice@example.com
To: bob@example.com
Subject: hello
Date: Mon, 1 Jan 2026 00:00:00 +0900
Labels: INBOX, Work/Todo
---
the body
`,
		},
		{
			name: "no labels omits the labels line",
			detail: &MessageDetail{
				ID:       "msg1",
				ThreadID: "thread1",
				URL:      "https://example.com/mail",
				From:     "alice@example.com",
				To:       "bob@example.com",
				Subject:  "hello",
				Date:     "Mon, 1 Jan 2026 00:00:00 +0900",
				Body:     "the body",
			},
			want: `ID: msg1
ThreadID: thread1
URL: https://example.com/mail
From: alice@example.com
To: bob@example.com
Subject: hello
Date: Mon, 1 Jan 2026 00:00:00 +0900
---
the body
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			if err := FormatMessageDetail(&buf, tt.detail, OutputFormatText); err != nil {
				t.Fatalf("FormatMessageDetail() error = %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("FormatMessageDetail() output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{name: "shorter than max", s: "abc", maxLen: 10, want: "abc"},
		{name: "exactly max", s: "abcde", maxLen: 5, want: "abcde"},
		{name: "longer than max", s: "abcdefghij", maxLen: 8, want: "abcde..."},
		{name: "empty string", s: "", maxLen: 5, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := truncate(tt.s, tt.maxLen); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}
