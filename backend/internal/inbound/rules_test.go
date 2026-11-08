package inbound

import "testing"

func TestRulesMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		from, subject, body string
		want                Class
	}{
		{"MAILER-DAEMON@mx.example", "Undeliverable: hi", "", ClassDSN},
		{"ada@acme.com", "please unsubscribe", "take me off this list", ClassUnsubscribe},
		{"ada@acme.com", "re: role", "this is spam", ClassComplaint},
		{"ada@acme.com", "Out of Office", "I am on vacation", ClassOutOfOffice},
		{"ada@acme.com", "re: role", "I'm interested in chatting", ClassInterested},
		{"ada@acme.com", "next step", "Can we set up an interview next week?", ClassInterview},
	}
	for _, tt := range tests {
		got, ok := Rules(tt.from, tt.subject, tt.body)
		if !ok || got != tt.want {
			t.Fatalf("%q: got %q ok=%v want %s", tt.body+tt.subject, got, ok, tt.want)
		}
	}
	if _, ok := Rules("ada@acme.com", "thanks", "let's chat next week"); ok {
		t.Fatal("plain reply should not match rules")
	}
}

func TestUnsubscribePolicyNeverReplyDraft(t *testing.T) {
	t.Parallel()
	for _, c := range []Class{ClassUnsubscribe, ClassComplaint, ClassDSN} {
		e := Decide(c, 1)
		if e.ReplyDraft {
			t.Fatalf("%s enqueued a reply draft", c)
		}
	}
	if Decide(ClassOther, 0.2).ReplyDraft {
		t.Fatal("low-confidence other must not draft")
	}
	if !Decide(ClassOther, 0.9).ReplyDraft {
		t.Fatal("high-confidence other may draft")
	}
	if !Decide(ClassInterested, 1).ReplyDraft {
		t.Fatal("interested should draft")
	}
}
