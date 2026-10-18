package config

import (
	"aura/logging"
	"context"
	"testing"
)

func TestValidateNotificationTemplate(t *testing.T) {
	_, ld := logging.CreateLoggingContext(context.Background(), "test")
	logAction := ld.AddAction("test", logging.LevelInfo)
	fallback := DefaultNotificationTemplates().DownloadQueue

	// A template that isn't set uses the built-in one
	empty := Config_CustomNotification{}
	validateNotificationTemplate(logAction, "DownloadQueue", TemplateTypeDownloadQueue, &empty, fallback)
	if empty != fallback {
		t.Errorf("empty template = %+v, want the built-in one", empty)
	}

	// A variable that doesn't exist puts the built-in title back, the message is kept
	custom := Config_CustomNotification{Enabled: true, Title: "{{NotAVariable}}", Message: "{{MediaItemTitle}} by {{SetCreator}}"}
	validateNotificationTemplate(logAction, "DownloadQueue", TemplateTypeDownloadQueue, &custom, fallback)
	if custom.Title != fallback.Title || custom.Message != "{{MediaItemTitle}} by {{SetCreator}}" || !custom.Enabled {
		t.Errorf("custom template = %+v", custom)
	}
}
