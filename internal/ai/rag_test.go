package ai

import "testing"

func TestRankUsesExcerpt(t *testing.T) {
	text := "Gin은 Go 언어의 HTTP 웹 프레임워크다.\n라우터와 미들웨어를 제공한다.\n\n데이터베이스는 이 문단에서만 말한다."
	got := Rank(text, "Gin은 무엇인가", 2)
	if len(got) == 0 || got[0].Index != 1 {
		t.Fatalf("%#v", got)
	}
	if SummarizeExtractive(text) == "" {
		t.Fatal("empty summary")
	}
}
