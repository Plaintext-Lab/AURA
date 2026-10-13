package notificationtmpl

import (
	"slices"
	"testing"
)

func TestWrappedAllowed(t *testing.T) {
	got := WrappedAllowed(EventTestNotification)
	want := []string{"{{MediaServerName}}", "{{MediaServerType}}", "{{Timestamp}}"}
	if !slices.Equal(got, want) {
		t.Errorf("WrappedAllowed = %v, want %v", got, want)
	}
	if len(WrappedAllowed("unknown")) != 0 {
		t.Error("an unknown event has no variables")
	}
}
