package loglet

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/uswitch/loglet/cmd/loglet/options"
)

type JournalEntryFilter interface {
	Ret() <-chan error
	Entries() <-chan *JournalEntry
}

type journalEntryFilter struct {
	ret            chan error
	entries        chan *JournalEntry
	includeFilters []map[string]string
	excludeFilters []map[string]string
}

func NewJournalEntryFilter(loglet *options.Loglet, unfilteredEntries <-chan *JournalEntry, done <-chan struct{}) (JournalEntryFilter, error) {
	ret := make(chan error, 2)
	filteredEntries := make(chan *JournalEntry)

	includeFilters, err := parseFilters(loglet.IncludeFilters)
	if err != nil {
		return nil, err
	}

	excludeFilters, err := parseFilters(loglet.ExcludeFilters)
	if err != nil {
		return nil, err
	}

	filter := &journalEntryFilter{
		ret:            ret,
		entries:        filteredEntries,
		includeFilters: includeFilters,
		excludeFilters: excludeFilters,
	}

	go filter.start(unfilteredEntries, done)

	return filter, nil
}

func (j *journalEntryFilter) Ret() <-chan error {
	return j.ret
}

func (j *journalEntryFilter) Entries() <-chan *JournalEntry {
	return j.entries
}

func (j *journalEntryFilter) start(unfilteredEntries <-chan *JournalEntry, done <-chan struct{}) {
	defer close(j.ret)
	defer close(j.entries)

	for {
		var entry *JournalEntry

		select {
		case <-done:
			return
		case entry = <-unfilteredEntries:
			if entry == nil {
				return
			}
		}

		// if there are any include filters then an entry has to match at least one,
		// however there need to be exclude filters before a match will exclude a message
		included := len(j.includeFilters) == 0 || matchesFilters(j.includeFilters, entry.Fields)
		excluded := len(j.excludeFilters) > 0 && matchesFilters(j.excludeFilters, entry.Fields)

		if included && !excluded {
			select {
			case <-done:
				return
			case j.entries <- entry:
				continue
			}
		}
	}
}

var filterRe = regexp.MustCompile("^([^=]+)=([^=]+)$")

func parseFilters(rawFilters []string) ([]map[string]string, error) {
	orFilters := []map[string]string{}

	for _, rawFilter := range rawFilters {
		andFilters := make(map[string]string)

		for _, rawAndFilter := range strings.Split(rawFilter, ",") {
			matches := filterRe.FindStringSubmatch(rawAndFilter)

			if len(matches) == 0 {
				return nil, fmt.Errorf("'%s' doesn't match expected pattern Key=Value", rawAndFilter)
			} else {
				andFilters[matches[1]] = matches[2]
			}
		}

		orFilters = append(orFilters, andFilters)
	}

	return orFilters, nil
}

func matchesFilters(orFilters []map[string]string, fields map[string]string) bool {

	for _, andFilters := range orFilters {
		matched := true

		for k, filterValue := range andFilters {
			fieldValue, ok := fields[k]

			if !ok || fieldValue != filterValue {
				matched = false
				break
			}
		}

		if matched {
			return true
		}
	}

	return false
}
