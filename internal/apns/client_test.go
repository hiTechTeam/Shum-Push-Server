package apns

import (
	"strings"
	"testing"
)

func TestPayloadNamesTheSenderAndEventWithoutContent(t *testing.T) {
	for kind, key := range map[string]string{
		"message":    "PUSH_NEW_MESSAGE_BODY",
		"invitation": "Новое приглашение",
		"reaction":   "PUSH_NEW_REACTION_BODY",
	} {
		payload := Payload(Notification{EventID: "event-1", Kind: kind, Sender: "Аня"})
		aps := payload["aps"].(map[string]any)
		alert := aps["alert"].(map[string]any)
		if alert["loc-key"] != key || alert["title"] != "Аня" {
			t.Errorf("%s: alert %v, want title Аня and loc-key %s", kind, alert, key)
		}
		if _, hasBadge := aps["badge"]; hasBadge {
			t.Errorf("%s: a push must not change the icon badge", kind)
		}
		shum := payload["shum"].(map[string]any)
		if shum["event_id"] != "event-1" || shum["kind"] != kind || len(shum) != 2 {
			t.Errorf("%s: unexpected routing data %v", kind, shum)
		}
	}
}

func TestSenderTitleCannotImpersonateOrOverflow(t *testing.T) {
	cases := map[string]string{
		"  Аня  ":                  "Аня",
		"Ann‮yna":                  "Annyna",
		"Line\nbreak\tand\x00zero": "Linebreakandzero",
		"⁦⁩  ":                     "Shum",
		"":                         "Shum",
		"👩‍💻 Dev":                  "👩‍💻 Dev",
	}
	for name, want := range cases {
		if got := senderTitle(name); got != want {
			t.Errorf("senderTitle(%q) = %q, want %q", name, got, want)
		}
	}
	if got := senderTitle(strings.Repeat("я", 100)); len([]rune(got)) != 64 {
		t.Errorf("long names are cut to 64 characters, got %d", len([]rune(got)))
	}
}
