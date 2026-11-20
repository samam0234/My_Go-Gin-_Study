# study17. 실시간과 확장

## ① 목표

글을 등록하면 로그인한 브라우저가 웹소켓으로 `post.created` 알림을 받게 한다.

## ② 📖 이론

HTTP 목록 조회는 클라이언트가 물어볼 때만 답을 준다. 알림은 서버가 먼저 말한다. 웹소켓은 그 통로다. Gin 핸들러가 HTTP 연결을 `gorilla/websocket`으로 업그레이드하고, `notify.Hub`가 연결된 클라이언트에 같은 JSON을 쓴다.

브라우저 웹소켓은 `Authorization` 헤더를 붙이기 어렵다. 그래서 이 학습 서버는 쿼리 `access_token`도 받는다. 토큰이 없거나 서명이 틀리면 업그레이드 전에 401이다. 토큰이 쿼리에 있으면 접근 로그와 브라우저 기록에 남을 수 있다. 로컬 학습에서만 쓰고, 운영에서는 짧은 수명과 로그 마스킹을 다시 본다.

파일 알림은 만들지 않는다. 업로드는 study13의 HTTP로 끝나고, 실시간은 새 글 이벤트만 보낸다. 허브는 프로세스 메모리다. 서버가 둘이면 다른 서버의 연결에는 알림이 가지 않는다.

비유하면 웹소켓은 카운터 옆 초인종이다. 주문서(HTTP)는 그대로 두고, 새 요리가 나오면 초인종만 울린다.

## ③ 🛠️ 실기

경로 `GET /ws?access_token=<JWT>`.

`internal/httpapi/extra.go` 의 `socket`:

```go
token := c.Query("access_token")
if token == "" {
	header := c.GetHeader("Authorization")
	token = strings.TrimPrefix(header, "Bearer ")
	if token == header {
		token = ""
	}
}
if _, err := auth.Parse(s.cfg.JWTSecret, token); err != nil {
	writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다.", nil)
	return
}
```

1. 쿼리 토큰을 먼저 보고, 없으면 Bearer를 본다.
2. 접두어가 아니면 토큰을 비운다.
3. 서명 실패는 업그레이드하지 않고 401 JSON이다.
4. `Upgrader`는 개발에서 localhost 출처를 허용하고, 운영에서는 `CORS_ORIGIN`만 허용한다. 출처 헤더가 없는 비브라우저 클라이언트는 허용한다.

허브 `internal/notify/hub.go` 는 연결마다 쓰기 잠금을 둔다. 한 연결에 두 고루틴이 동시에 쓰면 소켓이 깨지기 때문이다. 쓰기 실패 시 그 클라이언트를 제거한다.

글 등록이 성공한 뒤:

```go
note, _ := json.Marshal(gin.H{"type": "post.created", "data": postListJSON(post)})
s.hub.Broadcast(note)
```

본문은 목록 항목과 같다. 비밀번호나 본문 전체가 필요하지 않다.

테스트 `TestRateLimitAndSocket`이 `httptest` 서버에 다이얼하고, 글을 만든 뒤 2초 안에 `post.created` 를 읽는다.

```powershell
go test ./internal/httpapi -run TestRateLimitAndSocket -count=1
```

브라우저에서 보려면 개발자도구 콘솔보다, 이 테스트를 통과시키는 쪽이 재현이 분명하다. 화면 파일은 알림을 그리지 않고 HTTP로 목록을 다시 읽는다. 알림을 화면에 붙일 때는 `textContent`로 `data.title`만 보여 주면 된다.

## ④ 🔗 연동 포인트

토큰은 study7과 같다. 알림의 `data`는 study14 목록 항목과 같다. 화면이 알림을 받으면 `loadPosts()`를 다시 호출하면 캐시가 아직 남아 있을 수 있다. 등록 핸들러가 캐시를 지운 뒤에 브로드캐스트하므로, 다른 연결이 곧바로 목록을 다시 읽으면 MISS로 새 글을 본다. study15의 5초 캐시는 같은 프로세스의 쓰기가 이미 지운 상태다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 업그레이드 전에 401 | 토큰 누락 또는 만료 | `access_token` 또는 Bearer |
| 운영에서 연결이 바로 끊김 | 출처가 CORS 목록에 없음 | `CORS_ORIGIN`에 화면 출처 |
| 알림이 한 서버에만 감 | 허브가 메모리 | 학습 구성은 프로세스 하나 |
| 테스트가 2초 타임아웃 | 브로드캐스트가 등록 실패 뒤에 있음 | 201 이후에만 `Broadcast` |
| 소켓이 패닉 | 동시 쓰기 | 클라이언트별 뮤텍스 |

## ⑥ 직접 해보기

1. 토큰 없이 `/ws` 를 열어 401 JSON인지 본다.
2. `TestRateLimitAndSocket` 을 실행해 `post.created` 를 확인한다.
3. 알림 JSON에 `type`과 `data.title`만 있는지 테스트 로그로 본다.

## ⑦ ☐ 체크리스트

- [ ] `/ws` 는 유효한 JWT가 있어야 업그레이드된다
- [ ] 글 등록 성공 뒤에만 알림을 보낸다
- [ ] 알림 본문에 비밀번호와 해시가 없다
- [ ] 개발과 운영의 출처 검사가 다르다
- [ ] 허브 쓰기는 연결마다 잠긴다
- [ ] 웹소켓 테스트가 통과한다

## ⑧ 다음 단계

알림까지 붙였으면 study18에서 문서 질문과 MCP로 이 게시판을 모델에 읽히게 한다.
