package loglet

import (
	"fmt"
	"os"
	"os/signal"
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/uswitch/loglet/cmd/loglet/options"
)

func Run(loglet *options.Loglet) error {
	var returnErr error
	done := make(chan struct{})

	cursorState := NewCursorState(loglet.CursorFile)
	cursor, err := cursorState.Cursor()
	if err != nil {
		return fmt.Errorf("unable to read cursor state: %s", err)
	}

	journal := NewJournalFollower(cursor, done)

	filter, err := NewJournalEntryFilter(loglet, journal.Entries(), done)
	if err != nil {
		return fmt.Errorf("unable to create filter: %s", err)
	}

	transformer := NewJournalEntryTransformer(loglet, filter.Entries(), done)

	publisher, err := NewKinesisPublisher(loglet, transformer.Messages(), done)
	if err != nil {
		return fmt.Errorf("unable to create publisher: %s", err)
	}

	committer := NewCursorCommitter(cursorState, publisher.Published(), done)

	rets := merge(journal.Ret(), filter.Ret(), transformer.Ret(), publisher.Ret(), committer.Ret())

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	// wait for either sigint, or a process exiting prematurely
	log.Infof("started")
	select {
	case <-sigint:
	case returnErr = <-rets:
		if returnErr != nil {
			log.Errorf("service: process exited prematurely with error: %s", returnErr)
		} else {
			log.Debugf("service: process exited without error")
		}
	}

	log.Infof("exiting")

	// we're done
	close(done)

	// wait for remaining processes
	for err := range rets {
		if err != nil {
			log.Errorf("service: process returned with error: %s", err)
		}
	}

	return returnErr
}

func merge(cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	output := func(c <-chan error) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}

	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
