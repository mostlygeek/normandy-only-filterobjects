package tools

import (
	"io"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
)

type RecordHandler func(record []byte) error

// WalkAPI walks through API result pages until
// there are no more pages or handler returns an error
func WalkAPI(next string, handler RecordHandler) error {
	for {
		if next == "" {
			break
		}

		body, err := Get(next)
		if err != nil {
			return errors.Wrapf(err, "Failed to walk url: %s", next)
		}

		next, err = jsonparser.GetString(body, "next")
		if err != nil {
			return io.EOF
		}

		callHandler := true

		jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if !callHandler {
				return
			}

			if err := handler(value); err != nil {
				// clear out next so we don't parse anymore
				next = ""
				callHandler = false
			}
		}, "results")
	}

	return io.EOF
}
