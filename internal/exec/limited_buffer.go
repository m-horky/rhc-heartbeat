package exec

import "bytes"

const maxOutputSize = 1024 * 1024

// limitedBuffer captures one process stream up to its configured limit.
type limitedBuffer struct {
	limit     int
	buffer    bytes.Buffer
	truncated bool
}

// Write captures as many bytes as remain and discards excess bytes while draining the stream.
func (buffer *limitedBuffer) Write(data []byte) (int, error) {
	remaining := buffer.limit - buffer.buffer.Len()

	captured := len(data)
	if captured > remaining {
		captured = remaining
		buffer.truncated = true
	}

	if captured > 0 {
		_, _ = buffer.buffer.Write(data[:captured])
	}

	return len(data), nil
}
