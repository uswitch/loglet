package loglet

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/uswitch/loglet/cmd/loglet/options"
	"github.com/uswitch/loglet/transformers"
	"github.com/uswitch/loglet/types"
)

type EncodedMessage struct {
	Cursor  string
	Message []byte
}

type JournalEntryTransformer interface {
	Ret() <-chan error
	Messages() <-chan *EncodedMessage
}

type journalEntryTransformer struct {
	ret          chan error
	messages     chan *EncodedMessage
	transformers []types.Transformer
}

func NewJournalEntryTransformer(loglet *options.Loglet, entries <-chan *JournalEntry, done <-chan struct{}) JournalEntryTransformer {
	ret := make(chan error)
	messages := make(chan *EncodedMessage)

	var ts []types.Transformer

	if loglet.DefaultFields != nil {
		ts = append(ts, transformers.NewDefaultFields(loglet.DefaultFields))
	}

	converter := &journalEntryTransformer{
		ret:          ret,
		messages:     messages,
		transformers: ts,
	}
	go converter.convert(entries, done)

	return converter
}

func (c *journalEntryTransformer) Ret() <-chan error {
	return c.ret
}

func (c *journalEntryTransformer) Messages() <-chan *EncodedMessage {
	return c.messages
}

func (c *journalEntryTransformer) convert(entries <-chan *JournalEntry, done <-chan struct{}) {
	defer close(c.ret)
	defer close(c.messages)

	for {
		var entry *JournalEntry

		select {
		case <-done:
			return
		case entry = <-entries:
			if entry == nil {
				return
			}
		}

		m, err := c.convertToLogstashMessage(entry)
		if err != nil {
			c.ret <- fmt.Errorf("transformer: unable to convert journal entry to logstash message: %s", err)
			return
		}

		select {
		case <-done:
			return
		case c.messages <- m:
			continue
		}
	}

}

func (c *journalEntryTransformer) convertToLogstashMessage(entry *JournalEntry) (*EncodedMessage, error) {
	fields := readFields(entry)

	timestamp, err := readTime(fields)
	if err != nil {
		return nil, err
	}

	fields["@timestamp"] = timestamp.Format("2006-01-02T15:04:05.000Z")

	logMessage := &types.LogMessage{
		Fields: fields,
	}

	for _, t := range c.transformers {
		t.Transform(logMessage)
	}

	m, err := json.Marshal(logMessage.Fields)
	if err != nil {
		return nil, fmt.Errorf("transformer: unable to encode message as json: %v", err)
	}

	return &EncodedMessage{
		Cursor:  entry.Cursor,
		Message: m,
	}, nil
}

func readFields(entry *JournalEntry) map[string]interface{} {
	fields := make(map[string]interface{})

	for key, val := range entry.Fields {
		formattedKey := strings.ToLower(strings.TrimLeft(key, "_"))
		switch formattedKey {
		case "cap_effective":
		case "cmdline":
		//case "cursor":
		case "exe":
		case "machine_id":
		case "monotonic_timestamp":
		case "source_monotonic_timestamp":
		case "source_realtime_timestamp":
		case "syslog_facility":
		case "syslog_identifier":
		case "transport":
			continue
		default:
			fields[formattedKey] = val
		}
	}

	return fields
}

func readTime(fields map[string]interface{}) (*time.Time, error) {
	timeField, ok := fields["realtime_timestamp"].(string)
	if !ok {
		return nil, fmt.Errorf("timestamp field not found")
	}

	usSinceEpoch, err := strconv.Atoi(timeField)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert '%s' to integer: %s", timeField, err)
	}

	ts := time.Unix(int64(usSinceEpoch/1000000), int64(usSinceEpoch%1000000000)).UTC()

	return &ts, nil
}
