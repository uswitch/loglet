package loglet

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

type JournalEntry struct {
	Cursor string
	Fields map[string]string
}

type JournalFollower interface {
	Ret() <-chan error
	Entries() <-chan *JournalEntry
}

type journalFollower struct {
	ret     chan error
	entries chan *JournalEntry
}

func NewJournalFollower(cursor string, done <-chan struct{}) JournalFollower {
	ret := make(chan error, 2)
	entries := make(chan *JournalEntry)

	follower := &journalFollower{
		ret:     ret,
		entries: entries,
	}
	go follower.start(cursor, done)

	return follower
}

func (j *journalFollower) Ret() <-chan error {
	return j.ret
}

func (j *journalFollower) Entries() <-chan *JournalEntry {
	return j.entries
}

func (j *journalFollower) start(cursor string, done <-chan struct{}) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		j.ret <- fmt.Errorf("journal: unable to stat stdin: %s", err)
		return

	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		go j.startStdin(done)
	} else {
		go j.startJournalctl(cursor, done)
	}

}

func (j *journalFollower) startStdin(done <-chan struct{}) {
	defer close(j.entries)
	defer close(j.ret)

	err := j.sendEntries(os.Stdin, done)
	if err != nil {
		j.ret <- fmt.Errorf("journal: unable to push entries: %s", err)
		return
	}
}

func (j *journalFollower) startJournalctl(cursor string, done <-chan struct{}) {
	defer close(j.ret)

	var wg sync.WaitGroup
	var cmd *exec.Cmd
	followerDone := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(j.entries)
		defer close(followerDone)

		args := []string{"--output", "export", "--follow"}
		if cursor != "" {
			args = append(args, "--no-tail", "--after-cursor", cursor)
		} else {
			args = append(args, "--lines=0")
		}

		cmd = exec.Command("journalctl", args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			j.ret <- fmt.Errorf("journal: could not initialize pipe for journalctl output: %s", err)
			return
		}

		err = cmd.Start()
		if err != nil {
			j.ret <- fmt.Errorf("journal: could not execute journalctl: %s", err)
			return
		}

		err = j.sendEntries(stdout, done)
		if err != nil {
			j.ret <- fmt.Errorf("journal: unable to push entries: %s", err)
			return
		}
	}()

	// kill journalctl process when done
	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-done:
			if cmd != nil {
				err := cmd.Process.Kill()
				if err != nil {
					j.ret <- fmt.Errorf("journal: unable to kill journalctl process: %s", err)
				}
			}
			return

		case <-followerDone:
			return
		}
	}()

	// close return channel when all done
	wg.Wait()
}

func (j *journalFollower) sendEntries(output io.ReadCloser, done <-chan struct{}) error {
	defer output.Close()

	reader := bufio.NewReader(output)

	for {
		fields, err := decodeEntry(reader)
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return fmt.Errorf("journal: could not decode entry: %s\n", err)
		}

		cursor, ok := fields["__CURSOR"]
		if !ok {
			return fmt.Errorf("journal: __CURSOR field missing")
		}

		entry := &JournalEntry{
			Fields: fields,
			Cursor: cursor,
		}

		select {
		case <-done:
			return nil
		case j.entries <- entry:
			continue
		}
	}
}

// Decode a journald entry in the 'export' format as described here:
// https://www.freedesktop.org/wiki/Software/systemd/export/
func decodeEntry(reader *bufio.Reader) (map[string]string, error) {
	fields := make(map[string]string)

	for {
		name, err := decodeFieldName(reader)
		switch {
		case err == io.EOF && len(fields) == 0:
			return nil, err
		case err != nil:
			return nil, fmt.Errorf("decoding field name: %s", err)
		}

		if name == nil {
			break
		}

		value, err := decodeFieldValue(reader)
		if err != nil {
			return nil, fmt.Errorf("decoding field value: %s, fields=%+v", err, fields)
		}
		fields[*name] = *value
	}

	b, err := reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("reading next entry: %s", err)
	}
	if b != '\n' {
		return nil, fmt.Errorf("expected newline, read=%x", b)
	}

	return fields, nil
}

func decodeFieldName(reader *bufio.Reader) (*string, error) {
	var buf bytes.Buffer

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		switch b {
		case '=':
			fallthrough
		case '\n':
			err := reader.UnreadByte()
			if err != nil {
				return nil, fmt.Errorf("unreading end of binary field: %s", err)
			}
			if buf.Len() > 0 {
				s := buf.String()
				return &s, nil
			} else {
				return nil, nil
			}
		default:
			err := buf.WriteByte(b)
			if err != nil {
				return nil, fmt.Errorf("writing to buffer: %s", err)
			}
		}
	}
}

func decodeFieldValue(reader *bufio.Reader) (*string, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	switch b {
	case '=':
		data, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("reading text field: %s", err)
		}
		s := string(data[:len(data)-1])
		return &s, nil
	case '\n':
		var size int64
		err := binary.Read(reader, binary.LittleEndian, &size)
		if err != nil {
			return nil, fmt.Errorf("reading binary field length: %s", err)
		}

		data := make([]byte, size)
		nread := int64(0)
		for {
			if nread == size {
				break
			}

			n, err := reader.Read(data[nread:])
			if err != nil {
				return nil, fmt.Errorf("reading binary field, nread=%d, size=%d: %s", nread, size, err)
			}

			nread = nread + int64(n)
		}
		r, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil, err
			} else {
				return nil, fmt.Errorf("reading newline after binary field: %s", err)
			}
		}
		if b != '\n' {
			return nil, fmt.Errorf("expected newline after binary field, read=%d", r)
		}

		s := string(data)
		return &s, nil
	default:
		return nil, fmt.Errorf("read unexpected rune, read=%x", b)
	}
}
