package loglet

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

type CursorState interface {
	Cursor() (string, error)
	Commit(cursor string) error
}

type cursorState struct {
	filename string
}

func NewCursorState(filename string) CursorState {
	return &cursorState{
		filename: filename,
	}
}

func (s *cursorState) Cursor() (string, error) {
	bytes, err := ioutil.ReadFile(s.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", err
	}
	return string(bytes), nil
}

func (s *cursorState) Commit(cursor string) error {
	err := ioutil.WriteFile(s.filename, []byte(cursor), 0644)
	if err != nil {
		return err
	}
	return nil
}

type CursorCommitter interface {
	Ret() <-chan error
}

type cursorCommitter struct {
	ret   chan error
	state CursorState
}

func NewCursorCommitter(state CursorState, cursors <-chan string, done <-chan struct{}) *cursorCommitter {
	ret := make(chan error, 1)

	committer := &cursorCommitter{
		ret:   ret,
		state: state,
	}
	go committer.loop(cursors, done)

	return committer
}

func (c *cursorCommitter) Ret() <-chan error {
	return c.ret
}

func (c *cursorCommitter) loop(cursors <-chan string, done <-chan struct{}) {
	defer close(c.ret)

	var lastPublishedCursor string

	timer := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-done:
			return

		case cursor, ok := <-cursors:
			if !ok {
				return
			}
			lastPublishedCursor = cursor

		case <-timer.C:
			err := c.state.Commit(lastPublishedCursor)
			if err != nil {
				c.ret <- fmt.Errorf("committer: unable to commit cursor state: %v", err)
				return
			}
		}
	}

}
