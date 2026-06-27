package compress

import (
	"fmt"
	"strings"
)

type NativeTextLogCompressor struct{}

func (NativeTextLogCompressor) Compress(block Block) (Proposal, error) {
	switch block.ContentType {
	case ContentLog:
		return Proposal{Text: compressRepeatedLines(block.Text)}, nil
	case ContentText:
		return Proposal{Text: compressRepeatedParagraphs(block.Text)}, nil
	default:
		return Proposal{Text: block.Text}, nil
	}
}

func compressRepeatedLines(text string) string {
	lines := strings.Split(text, "\n")
	return strings.Join(collapseAdjacent(lines, func(line string) string {
		return strings.TrimSpace(line)
	}, "identical log lines"), "\n")
}

func compressRepeatedParagraphs(text string) string {
	paragraphs := strings.Split(text, "\n\n")
	return strings.Join(collapseAdjacent(paragraphs, func(paragraph string) string {
		return strings.Join(strings.Fields(paragraph), " ")
	}, "identical text blocks"), "\n\n")
}

func collapseAdjacent(items []string, key func(string) string, label string) []string {
	if len(items) == 0 {
		return nil
	}

	collapsed := make([]string, 0, len(items))
	for i := 0; i < len(items); {
		item := items[i]
		itemKey := key(item)
		repeats := 1
		for i+repeats < len(items) && itemKey != "" && key(items[i+repeats]) == itemKey {
			repeats++
		}

		collapsed = append(collapsed, item)
		if repeats > 1 {
			collapsed = append(collapsed, fmt.Sprintf("[repeated %d %s omitted]", repeats-1, label))
		}
		i += repeats
	}
	return collapsed
}
