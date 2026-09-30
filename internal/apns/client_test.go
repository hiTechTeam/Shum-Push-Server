package apns

import "testing"

func TestPayloadNamesTheEventWithoutItsContent(t *testing.T) {
	for kind, key := range map[string]string{
		"message":    "PUSH_NEW_MESSAGE_BODY",
		"invitation": "PUSH_NEW_MESSAGE_BODY",
		"reaction":   "PUSH_NEW_REACTION_BODY",
	} {
		payload := Payload(Notification{EventID: "event-1", Kind: kind})
		alert := payload["aps"].(map[string]any)["alert"].(map[string]any)
		if alert["loc-key"] != key {
			t.Errorf("%s: loc-key %v, want %s", kind, alert["loc-key"], key)
		}
		if _, hasBadge := payload["aps"].(map[string]any)["badge"]; hasBadge {
			t.Errorf("%s: a push must not change the icon badge", kind)
		}
		shum := payload["shum"].(map[string]any)
		if shum["event_id"] != "event-1" || shum["kind"] != kind || len(shum) != 2 {
			t.Errorf("%s: unexpected routing data %v", kind, shum)
		}
	}
}
