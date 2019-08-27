// +build gofuzz

package uploader

func FuzzParseParams(data []byte) int {
	s := string(data)
	_, err := parseParams(s, s, s)
	if err != nil {
		return 0
	}
	return 1
}
