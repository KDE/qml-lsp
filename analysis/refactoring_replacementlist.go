package analysis

import "sort"

type replaceSpan struct {
	start uint32
	end   uint32
	with  string
}

type replacementlist []replaceSpan

func (replacements replacementlist) applyTo(sdata string) string {
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].start < replacements[j].start
	})

	offset := 0
	for _, replacement := range replacements {
		head := sdata[0 : int(replacement.start)-offset]
		tail := sdata[int(replacement.end)-offset:]

		sdata = head
		sdata = sdata + replacement.with
		sdata = sdata + tail

		diff := (int(replacement.end) - int(replacement.start)) - len(replacement.with)
		offset += diff
	}

	return sdata
}
