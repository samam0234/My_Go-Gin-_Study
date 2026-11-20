# study4. 라우팅과 요청 처리

## ① 목표

목록, 검색 쿼리, 숫자 경로, 잘못된 id, 없는 경로를 Gin 라우터에서 상태 코드와 함께 나눈다.

## ② 📖 이론

요청은 메서드와 경로로 도착한다. `GET /api/posts` 와 `GET /api/posts/2` 는 다른 핸들러다. `:id` 는 그 위치의 글자를 `c.Param("id")` 로 꺼내는 칸이다.

쿼리는 `?q=첫 글&page=1` 처럼 경로 뒤에 붙는다. `c.Query("q")` 가 값을 준다. 없으면 빈 문자열이다.

처리 순서:

1. 미들웨어가 요청 ID, 복구, CORS를 준비한다.
2. 라우터가 메서드와 경로를 맞춘다.
3. 핸들러가 경로 값을 숫자로 바꾸고, 실패하면 400 `BAD_REQUEST` 다.
4. 숫자는 맞지만 자료가 없으면 404 `NOT_FOUND` 다.
5. 맞는 경로 자체가 없으면 `NoRoute`가 404를 만든다.

공통 스펙의 글 경로는 `/api/posts`, `/api/posts/{id}` 다. 건강 체크만 접두어 없이 `/health` 다. 프론트의 화면 주소와 API 주소를 같은 문자열로 두면, 브라우저는 HTML을 기대하는데 JSON이 오거나 그 반대가 된다. 이 서버는 화면을 `/` 에 두고 API는 `/api` 아래에 둔다.

비유하면 경로는 건물 호수이고, 쿼리는 방문 메모다. 호수가 글자면 안내 데스크가 "숫자를 적어 주세요"(400)라고 하고, 호수는 숫자인데 방이 없으면 "없는 방"(404)이라고 한다.

## ③ 🛠️ 실기

`demo/step04/main.go`

```go
r.GET("/api/posts", func(c *gin.Context) {
	q := c.Query("q")
	out := make([]post, 0)
	for _, item := range posts {
		if q == "" || item.Title == q {
			out = append(out, item)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
})
r.GET("/api/posts/:id", func(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": "id는 숫자여야 합니다."}})
		return
	}
	for _, item := range posts {
		if item.ID == id {
			c.JSON(http.StatusOK, gin.H{"data": item})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "없습니다."}})
})
r.NoRoute(func(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "경로를 찾을 수 없습니다."}})
})
```

1. 목록은 쿼리 `q`가 비어 있으면 전부, 있으면 제목이 같은 것만 담는다.
2. `make([]post, 0)` 는 빈 배열로 JSON `[]` 가 되게 한다. nil이면 `null` 이다.
3. 상세는 경로의 `id`를 정수로 바꾼다.
4. 변환 실패나 1 미만은 400이다. 스펙도 숫자가 아닌 id를 `BAD_REQUEST`로 둔다.
5. `return`이 없으면 아래에 404를 또 쓴다.
6. 일치하는 글이 있으면 200으로 끝내고 `return` 한다.
7. 없으면 404다.
8. 등록되지 않은 경로는 `NoRoute`가 같은 오류 모양으로 답한다.

실제 서버의 목록은 더 많은 쿼리를 받는다. `internal/httpapi/posts.go` 의 `listQuery` 가 `q`, `page`, `limit`, `sort` 를 읽는다.

- `page` 기본 1. 1 미만은 400.
- `limit` 기본 10. 50을 넘으면 50으로 줄인다.
- `sort` 는 `createdAt`, `-createdAt`, `-commentCount` 만 허용한다. 다른 문자열은 SQL에 붙이지 않고 400이다.

페이지 이동은 화면의 이전/다음 버튼이 `state.page`를 바꾸고 `GET /api/posts?page=` 를 다시 호출하는 일이다. 서버는 위치를 기억하지 않는다.

테스트에서 한글과 공백이 있는 쿼리는 `url.QueryEscape`로 넣어야 한다. 요청 줄에 공백이 그대로 있으면 HTTP 문법이 깨진다.

## ④ 🔗 연동 포인트

이 경로 이름이 프론트 `project/web/app.js` 의 `api("/api/posts?page=" + state.page + ...)` 와 같다. 한쪽만 `/posts`로 바꾸면 화면은 404 JSON을 받는다. study5는 이 목록 경로에 `POST`를 더해 본문을 받는다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| `/api/posts/abc` 가 500 | 변환 오류를 무시하고 조회 | `Atoi` 실패를 400으로 끝낸다 |
| 빈 목록이 `null` | nil 슬라이스 | 길이 0 슬라이스를 만든다 |
| 쿼리에 글이 안 나옴 | 인코딩되지 않은 공백 | 클라이언트가 `encodeURIComponent` 를 쓴다 |
| 모든 404가 HTML | `NoRoute` 없이 Gin 기본 동작 | JSON `NoRoute`를 등록한다 |
| `sort`에 SQL이 섞일 위험 | 쿼리 문자열을 `ORDER BY`에 연결 | 허용 목록만 `switch`로 고른다 |
| 화면 주소와 API가 겹침 | `/posts`를 둘 다 씀 | 화면은 `/`, API는 `/api/posts` |

## ⑥ 직접 해보기

1. `/api/posts?q=` 와 제목 전체를 넣은 검색을 비교한다.
2. `/api/posts/0` 과 `/api/posts/99` 의 상태 코드가 다른지 본다.
3. 존재하지 않는 `/api/nowhere` 의 `error.code` 가 `NOT_FOUND` 인지 본다.

## ⑦ ☐ 체크리스트

- [ ] `GET /api/posts` 와 `GET /api/posts/:id` 를 나눴다
- [ ] 쿼리 `q`로 목록을 걸렀다
- [ ] 숫자가 아닌 id는 400 `BAD_REQUEST` 다
- [ ] 없는 글과 없는 경로는 404 `NOT_FOUND` 다
- [ ] 정렬 값은 허용 목록만 통과한다
- [ ] 화면의 페이지 번호가 쿼리 `page`로 다시 요청된다
- [ ] `go test ./demo/step04` 가 통과한다

## ⑧ 다음 단계

경로가 요청을 받으면 study5에서 JSON 본문을 구조체에 담고 검증한다.
