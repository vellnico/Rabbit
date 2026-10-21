// Package msg builds errors that the frontend translates.
//
// An error text "@key" or "@key|detail" is looked up in the frontend
// dictionary; any other text is shown as is.
package msg

import "fmt"

type coded struct {
	key    string
	detail string
	err    error
}

func (c *coded) Error() string {
	s := "@" + c.key
	if c.detail != "" {
		s += "|" + c.detail
	} else if c.err != nil {
		s += "|" + c.err.Error()
	}
	return s
}

func (c *coded) Unwrap() error { return c.err }

// E returns a translatable error. detail is optional and shown verbatim.
func E(key string, detail ...any) error {
	c := &coded{key: key}
	if len(detail) > 0 {
		c.detail = fmt.Sprint(detail...)
	}
	return c
}

// Wrap returns a translatable error whose detail is err's text.
func Wrap(key string, err error) error {
	return &coded{key: key, err: err}
}
