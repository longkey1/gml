package gml

import (
	"encoding/base64"
	"reflect"
	"testing"

	"google.golang.org/api/gmail/v1"
)

// b64 encodes a body the way the Gmail API does (URL-safe base64).
func b64(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}

func TestParseFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldsStr string
		want      map[string]bool
	}{
		{
			name:      "single field",
			fieldsStr: "id",
			want:      map[string]bool{"id": true},
		},
		{
			name:      "multiple fields",
			fieldsStr: "id,subject,from",
			want:      map[string]bool{"id": true, "subject": true, "from": true},
		},
		{
			name:      "whitespace is trimmed",
			fieldsStr: " id , subject ",
			want:      map[string]bool{"id": true, "subject": true},
		},
		{
			name:      "lowercased",
			fieldsStr: "ID,Subject,THREADID",
			want:      map[string]bool{"id": true, "subject": true, "threadid": true},
		},
		{
			name:      "duplicates collapse",
			fieldsStr: "id,id,ID",
			want:      map[string]bool{"id": true},
		},
		{
			name:      "empty string yields empty key",
			fieldsStr: "",
			want:      map[string]bool{"": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseFields(tt.fieldsStr)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseFields(%q) = %v, want %v", tt.fieldsStr, got, tt.want)
			}
		})
	}
}

func TestExtractBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload *gmail.MessagePart
		want    string
	}{
		{
			name:    "nil payload",
			payload: nil,
			want:    "",
		},
		{
			name: "plain text in top-level part",
			payload: &gmail.MessagePart{
				MimeType: "text/plain",
				Body:     &gmail.MessagePartBody{Data: b64("hello plain")},
			},
			want: "hello plain",
		},
		{
			name: "plain text preferred over html",
			payload: &gmail.MessagePart{
				MimeType: "multipart/alternative",
				Parts: []*gmail.MessagePart{
					{
						MimeType: "text/html",
						Body:     &gmail.MessagePartBody{Data: b64("<p>html</p>")},
					},
					{
						MimeType: "text/plain",
						Body:     &gmail.MessagePartBody{Data: b64("plain")},
					},
				},
			},
			want: "plain",
		},
		{
			name: "html fallback when no plain part",
			payload: &gmail.MessagePart{
				MimeType: "multipart/alternative",
				Parts: []*gmail.MessagePart{
					{
						MimeType: "text/html",
						Body:     &gmail.MessagePartBody{Data: b64("<p>html only</p>")},
					},
				},
			},
			want: "<p>html only</p>",
		},
		{
			name: "nested multipart",
			payload: &gmail.MessagePart{
				MimeType: "multipart/mixed",
				Parts: []*gmail.MessagePart{
					{
						MimeType: "multipart/alternative",
						Parts: []*gmail.MessagePart{
							{
								MimeType: "text/plain",
								Body:     &gmail.MessagePartBody{Data: b64("nested plain")},
							},
						},
					},
					{
						MimeType: "application/pdf",
						Body:     &gmail.MessagePartBody{Data: b64("binary")},
					},
				},
			},
			want: "nested plain",
		},
		{
			name: "main body fallback without matching mime type",
			payload: &gmail.MessagePart{
				MimeType: "text/x-custom",
				Body:     &gmail.MessagePartBody{Data: b64("raw body")},
			},
			want: "raw body",
		},
		{
			name: "invalid base64 in matching part",
			payload: &gmail.MessagePart{
				MimeType: "text/plain",
				Body:     &gmail.MessagePartBody{Data: "!!!not-base64!!!"},
			},
			want: "",
		},
		{
			name: "empty payload",
			payload: &gmail.MessagePart{
				MimeType: "multipart/mixed",
			},
			want: "",
		},
		{
			name: "part with nil body is skipped",
			payload: &gmail.MessagePart{
				MimeType: "multipart/alternative",
				Parts: []*gmail.MessagePart{
					{MimeType: "text/plain"},
					{
						MimeType: "text/html",
						Body:     &gmail.MessagePartBody{Data: b64("<p>fallback</p>")},
					},
				},
			},
			want: "<p>fallback</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ExtractBody(tt.payload); got != tt.want {
				t.Errorf("ExtractBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildMessageInfo(t *testing.T) {
	t.Parallel()

	msg := &gmail.Message{
		Id:       "msg1",
		ThreadId: "thread1",
		Snippet:  "a snippet",
		LabelIds: []string{"INBOX", "Label_123"},
		Payload: &gmail.MessagePart{
			Headers: []*gmail.MessagePartHeader{
				{Name: "From", Value: "alice@example.com"},
				{Name: "To", Value: "bob@example.com"},
				{Name: "Subject", Value: "hello"},
				{Name: "Date", Value: "Mon, 1 Jan 2026 00:00:00 +0900"},
			},
		},
	}

	idx := newLabelIndex(map[string]string{
		"INBOX":     "INBOX",
		"Work/Todo": "Label_123",
	})

	tests := []struct {
		name      string
		msg       *gmail.Message
		fields    map[string]bool
		userEmail string
		idx       *LabelIndex
		want      MessageInfo
	}{
		{
			name:      "all fields",
			msg:       msg,
			fields:    ParseFields("id,threadid,url,from,to,subject,date,labels,snippet"),
			userEmail: "user@example.com",
			idx:       idx,
			want: MessageInfo{
				ID:       "msg1",
				ThreadID: "thread1",
				URL:      "https://mail.google.com/mail/?authuser=user@example.com#all/thread1",
				From:     "alice@example.com",
				To:       "bob@example.com",
				Subject:  "hello",
				Date:     "Mon, 1 Jan 2026 00:00:00 +0900",
				Labels:   []string{"INBOX", "Work/Todo"},
				Snippet:  "a snippet",
			},
		},
		{
			name:   "subset of fields",
			msg:    msg,
			fields: ParseFields("id,subject"),
			idx:    idx,
			want: MessageInfo{
				ID:      "msg1",
				Subject: "hello",
			},
		},
		{
			name:   "labels requested with nil index are omitted",
			msg:    msg,
			fields: ParseFields("labels"),
			idx:    nil,
			want:   MessageInfo{},
		},
		{
			name: "nil payload leaves header fields empty",
			msg: &gmail.Message{
				Id:       "msg2",
				ThreadId: "thread2",
			},
			fields: ParseFields("id,from,subject"),
			idx:    idx,
			want:   MessageInfo{ID: "msg2"},
		},
		{
			name:   "no fields selected",
			msg:    msg,
			fields: map[string]bool{},
			idx:    idx,
			want:   MessageInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildMessageInfo(tt.msg, tt.fields, tt.userEmail, tt.idx)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildMessageInfo() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
