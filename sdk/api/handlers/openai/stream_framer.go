package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func writeOpenAIStreamData(w io.Writer, chunk []byte) {
	trimmed := bytes.TrimSpace(chunk)
	if len(trimmed) == 0 {
		return
	}
	if bytes.HasPrefix(trimmed, []byte("data:")) || bytes.HasPrefix(trimmed, []byte("event:")) {
		_, _ = w.Write(chunk)
		if !bytes.HasSuffix(chunk, []byte("\n\n")) {
			_, _ = w.Write([]byte("\n\n"))
		}
		return
	}
	for _, payload := range splitConcatenatedJSONPayloads(trimmed) {
		_, _ = fmt.Fprintf(w, "data: %s\n\n", string(payload))
	}
}

func splitConcatenatedJSONPayloads(payload []byte) [][]byte {
	dec := json.NewDecoder(bytes.NewReader(payload))
	var out [][]byte
	for {
		var raw json.RawMessage
		err := dec.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return [][]byte{bytes.Clone(payload)}
		}
		raw = bytes.TrimSpace(raw)
		if len(raw) > 0 {
			out = append(out, bytes.Clone(raw))
		}
	}
	if len(out) == 0 {
		return [][]byte{bytes.Clone(payload)}
	}
	return out
}
