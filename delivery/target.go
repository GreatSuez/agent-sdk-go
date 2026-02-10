package delivery

import "strings"

// Target identifies where an agent response should be sent.
// It is transport-agnostic and can represent DevUI, API/webhook, Slack, Telegram, etc.
type Target struct {
	Channel     string            `json:"channel,omitempty"`
	Destination string            `json:"destination,omitempty"`
	ThreadID    string            `json:"threadId,omitempty"`
	UserID      string            `json:"userId,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Normalize trims values and returns nil when no routing info is provided.
func Normalize(in *Target) *Target {
	if in == nil {
		return nil
	}
	out := &Target{
		Channel:     strings.TrimSpace(in.Channel),
		Destination: strings.TrimSpace(in.Destination),
		ThreadID:    strings.TrimSpace(in.ThreadID),
		UserID:      strings.TrimSpace(in.UserID),
	}
	if len(in.Metadata) > 0 {
		out.Metadata = map[string]string{}
		for k, v := range in.Metadata {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			out.Metadata[key] = strings.TrimSpace(v)
		}
		if len(out.Metadata) == 0 {
			out.Metadata = nil
		}
	}
	if out.Channel == "" && out.Destination == "" && out.ThreadID == "" && out.UserID == "" && len(out.Metadata) == 0 {
		return nil
	}
	return out
}
