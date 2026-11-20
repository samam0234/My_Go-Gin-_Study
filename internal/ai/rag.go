package ai

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

type Excerpt struct {
	Index int     `json:"index"`
	Text  string  `json:"text"`
	Score float64 `json:"score"`
}

// Chunk 는 빈 줄과 길이를 기준으로 문서를 나눈다.
// 임베딩 서버 없이 동작하는 학습용 분할이다.
func Chunk(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var chunks []string
	var b strings.Builder
	flush := func() {
		part := strings.TrimSpace(b.String())
		b.Reset()
		if part != "" {
			chunks = append(chunks, part)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if b.Len() > 0 && b.Len()+len(line) > 400 {
			flush()
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	flush()
	return chunks
}

// Rank 는 질문과 겹치는 토큰이 많은 청크를 고른다.
// 한글은 단어와 함께 두 글자 묶음도 봐서 조사 차이를 조금 흡수한다.
func Rank(text, query string, top int) []Excerpt {
	if top < 1 {
		top = 3
	}
	queryTokens := tokens(query)
	var ranked []Excerpt
	for i, chunk := range Chunk(text) {
		score := overlap(queryTokens, tokens(chunk))
		if score <= 0 {
			continue
		}
		ranked = append(ranked, Excerpt{
			Index: i + 1,
			Text:  chunk,
			Score: math.Round(score*10000) / 10000,
		})
	}
	sort.SliceStable(ranked, func(a, b int) bool {
		return ranked[a].Score > ranked[b].Score
	})
	if len(ranked) > top {
		ranked = ranked[:top]
	}
	return ranked
}

func SummarizeExtractive(text string) string {
	chunks := Chunk(text)
	if len(chunks) == 0 {
		return ""
	}
	summary := chunks[0]
	if len(chunks) > 1 {
		summary += "\n" + chunks[1]
	}
	runes := []rune(summary)
	if len(runes) > 240 {
		return string(runes[:240]) + "…"
	}
	return summary
}

func tokens(s string) []string {
	s = strings.ToLower(s)
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	var out []string
	for _, field := range fields {
		out = append(out, field)
		rs := []rune(field)
		if len(rs) >= 2 && unicode.Is(unicode.Hangul, rs[0]) {
			for i := 0; i < len(rs)-1; i++ {
				out = append(out, string(rs[i:i+2]))
			}
		}
	}
	return out
}

func overlap(query, doc []string) float64 {
	if len(query) == 0 || len(doc) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(doc))
	for _, token := range doc {
		set[token] = struct{}{}
	}
	hit := 0
	for _, token := range query {
		if _, ok := set[token]; ok {
			hit++
		}
	}
	return float64(hit) / float64(len(query))
}
