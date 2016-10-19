package loglet

import (
	"github.com/uswitch/loglet/transformers"
	"github.com/uswitch/loglet/types"
	"testing"
)

func sampleMessage(k, v string) *types.LogMessage {
	return &types.LogMessage{
		Fields: map[string]interface{}{
			k: v,
		},
	}
}

func TestDefaultFields(t *testing.T) {
	message := sampleMessage("a", "hello")
	defaults := transformers.NewDefaultFields(map[string]string{"foo": "bar"})
	defaults.Transform(message)

	if message.Fields["a"] != "hello" {
		t.Error("expected message[a] to be hello, was", message.Fields["a"])
	}
	if message.Fields["foo"] != "bar" {
		t.Error("expected message[foo] to be bar, was", message.Fields["foo"])
	}

	fooMessage := sampleMessage("foo", "baz")
	defaults.Transform(fooMessage)
	if fooMessage.Fields["foo"] != "baz" {
		t.Error("shouldnt have overwritten foo value from baz original, was", fooMessage.Fields["foo"])
	}
}
