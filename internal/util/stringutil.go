package util

func CommonBase(strings ...string) string {
	if len(strings) == 0 {
		return ""
	}

	common, tail := strings[0], strings[1:]

	for _, s := range tail {
		common = commonBase(common, s)
	}

	return common
}

func commonBase(a string, b string) string {

	longer, shorter := longerAndShorter(a, b)
	breakIx := 0

	longerRunes := []rune(longer)
	for ix, char := range shorter {
		if longerRunes[ix] != char {
			break
		}
		breakIx = ix + 1
	}

	return string(longerRunes[:breakIx])
}

func longerAndShorter(a string, b string) (string, string) {
	if len(a) > len(b) {
		return a, b
	}
	return b, a
}
