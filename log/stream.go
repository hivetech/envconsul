/*
 * https://github.com/kvz/logstreamer
 */
package log

import (
	"bytes"
	"io"
	"strings"
)

type Logstream struct {
	Logger *IronLogger
	buf    *bytes.Buffer
	output string
	prefix string
	// For clearer output
	colorPrefix string
	colorReset  string
}

func NewLogstream(logger *IronLogger, output string, prefix string) *Logstream {
	return &Logstream{
		Logger:      logger,
		buf:         bytes.NewBuffer([]byte("")),
		output:      output,
		prefix:      prefix,
		colorPrefix: "\x1b[32m",
		colorReset:  "\x1b[0m",
	}
}

func (l *Logstream) Write(p []byte) (n int, err error) {
	if n, err = l.buf.Write(p); err != nil {
		return
	}
	err = l.OutputLines()
	return
}

func (l *Logstream) Close() error {
	l.Flush()
	l.buf = bytes.NewBuffer([]byte(""))
	return nil
}

func (l *Logstream) Flush() error {
	var p []byte
	if _, err := l.buf.Read(p); err != nil {
		return err
	}

	l.out(string(p))
	return nil
}

func (l *Logstream) OutputLines() (err error) {
	for {
		line, err := l.buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		l.out(line)
	}

	return nil
}

func (l *Logstream) out(str string) (err error) {

	// Build colorful output and remove trailing carriage return
	str = strings.TrimSpace(l.colorPrefix + l.prefix + l.colorReset + " " + str)
	if l.output == "stdout" {
		l.Logger.Info(str)
	} else if l.output == "stderr" {
		l.Logger.Error(str)
	} else {
		l.Logger.Debug(str)
	}

	return nil
}
