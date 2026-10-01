package host

// shellPipeInputNormalizer gives Windows command interpreters the CRLF line
// endings they expect when stdin is a pipe rather than a console/ConPTY.
type shellPipeInputNormalizer struct {
	previousCR bool
}

func (n *shellPipeInputNormalizer) Normalize(input []byte) []byte {
	output := make([]byte, 0, len(input)+1)
	for _, value := range input {
		if n.previousCR {
			n.previousCR = false
			if value == '\n' {
				continue
			}
		}
		if value == '\r' {
			output = append(output, '\r', '\n')
			n.previousCR = true
			continue
		}
		if value == '\n' {
			output = append(output, '\r', '\n')
			continue
		}
		output = append(output, value)
	}
	return output
}
