package compress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type NativeCompressor struct{}

func (NativeCompressor) Compress(block Block) (Proposal, error) {
	switch block.ContentType {
	case ContentText, ContentLog:
		return NativeTextLogCompressor{}.Compress(block)
	case ContentJSON, ContentDiff, ContentCode:
		return NativeStructuredCompressor{}.Compress(block)
	default:
		return Proposal{Text: block.Text}, nil
	}
}

type NativeStructuredCompressor struct{}

func (NativeStructuredCompressor) Compress(block Block) (Proposal, error) {
	switch block.ContentType {
	case ContentJSON:
		return Proposal{Text: compactJSON(block.Text)}, nil
	case ContentDiff:
		return Proposal{Text: compressDiff(block.Text)}, nil
	case ContentCode:
		return Proposal{Text: compressCode(block.Text)}, nil
	default:
		return Proposal{Text: block.Text}, nil
	}
}

func compactJSON(text string) string {
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, []byte(text)); err != nil {
		return text
	}
	return compacted.String()
}

func compressDiff(text string) string {
	lines := strings.Split(text, "\n")
	compressed := make([]string, 0, len(lines))
	omitted := 0
	flushOmitted := func() {
		if omitted > 0 {
			if omitted > 1 {
				compressed = append(compressed, fmt.Sprintf("[%d diff context lines omitted]", omitted))
			}
			omitted = 0
		}
	}

	for _, line := range lines {
		if keepDiffLine(line) {
			flushOmitted()
			compressed = append(compressed, line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		omitted++
	}
	flushOmitted()
	return strings.Join(compressed, "\n")
}

func keepDiffLine(line string) bool {
	return strings.HasPrefix(line, "diff --git ") ||
		strings.HasPrefix(line, "index ") ||
		strings.HasPrefix(line, "--- ") ||
		strings.HasPrefix(line, "+++ ") ||
		strings.HasPrefix(line, "@@ ") ||
		strings.HasPrefix(line, "+") ||
		strings.HasPrefix(line, "-")
}

func compressCode(text string) string {
	lines := strings.Split(text, "\n")
	collapsed := collapseAdjacent(lines, func(line string) string {
		if strings.TrimSpace(line) == "" {
			return ""
		}
		return strings.TrimSpace(line)
	}, "identical code lines")
	return strings.Join(collapseBlankRuns(collapsed), "\n")
}

func collapseBlankRuns(lines []string) []string {
	collapsed := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if blank {
				continue
			}
			blank = true
			collapsed = append(collapsed, line)
			continue
		}
		blank = false
		collapsed = append(collapsed, line)
	}
	return collapsed
}
