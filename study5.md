# study5. 폼, 입력, 검증

## ① 목표

JSON 본문을 구조체에 묶고, 빠진 값과 길이 오류를 400으로 돌려주며, 같은 Idempotency-Key 의 재전송은 글을 한 번만 만들게 한다.

## ② 📖 이론

사용자가 보내는 값은 믿기 전에 검사한다. Gin은 `ShouldBindJSON`이 JSON을 구조체 포인터에 넣고, `binding` 태그를 검사한다.

```go
type postIn struct {
	Title string `json:"title" binding:"required,min=1,max=200"`
	Body  string `json:"body" binding:"required,min=1,max=2000"`
}
```

- `required`: 필드가 없거나 제로 값이면 실패
- `min`, `max`: 문자열 길이
- `email`: 메일 형식
- `dive`: 슬라이스 안쪽 항목도 검사

실패하면 공통 스펙은 422가 아니라 400 `VALIDATION_ERROR` 다. 본문은 아래 모양이다.

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "입력값을 확인하세요.",
    "fields": { "title": ["1~200자로 입력하세요."] },
    "requestId": "..."
  }
}
```

JSON 문법 자체가 깨지면 `fields` 없이 400 `BAD_REQUEST` 다. 어떤 API는 검증 실패를 422로 주기도 한다. 이 저장소의 화면은 400을 검증 실패로 보여주고, 422를 만나면 같은 메시지 칸에 넣으면 된다.

중복 제출은 버튼을 두 번 누르거나 네트워크가 재시도할 때 생긴다. 화면은 요청 중에 버튼을 끄고, 서버는 `Idempotency-Key` 가 같으면 처음 응답을 다시 준다.

미들웨어는 핸들러 앞에서 공통 일을 한다. 이 단계의 작은 예제는 로거, 패닉 복구, CORS 헤더를 엔진에 붙인다. 실제 서버도 `gin.Logger`에 해당하는 접근 로그, `gin.Recovery`, `CORS_ORIGIN` 허용 목록을 둔다.

비유하면 바인딩은 신청서를 칸에 옮겨 적는 일이고, 검증은 빈칸과 글자 수를 세는 일이다. 같은 접수 번호를 다시 가져오면 창구는 새 서류를 만들지 않고 아까 준 접수증을 다시 보여 준다.

## ③ 🛠️ 실기

`demo/step05/main.go` 를 본다.

```go
r := gin.New()
r.Use(gin.Logger(), gin.Recovery(), allowLocalhost)
r.POST("/api/posts", func(c *gin.Context) {
	key := c.GetHeader("Idempotency-Key")
	if key != "" {
		mu.Lock()
		if id, ok := seen[key]; ok {
			mu.Unlock()
			c.Header("Idempotency-Replayed", "true")
			c.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": id}})
			return
		}
		mu.Unlock()
	}
	var in postIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "입력값을 확인하세요."}})
		return
	}
	// id를 올리고, key가 있으면 seen에 저장한 뒤 201
})
```

1. 로거와 복구, CORS 미들웨어를 모든 요청에 건다.
2. `allowLocalhost`는 `Access-Control-Allow-Origin`을 `http://localhost:5173` 로 두고, `OPTIONS`는 204로 끝낸다.
3. 헤더에서 중복 방지 키를 읽는다.
4. 맵은 여러 요청이 동시에 건드리므로 `sync.Mutex`로 감싼다.
5. 이미 있는 키면 같은 id와 `Idempotency-Replayed: true` 를 주고 끝낸다.
6. 잠금을 풀기 전에 return 하면 다음 요청이 영원히 기다린다. 그래서 return 전에 `Unlock` 한다.
7. `ShouldBindJSON(&in)` 에 포인터를 넘긴다.
8. 실패는 400 `VALIDATION_ERROR` 다.

실제 서버 `internal/httpapi/bind.go` 의 `bindJSON`은 오류 종류를 나눈다.

- 본문이 1MB를 넘으면 413 `PAYLOAD_TOO_LARGE`
- `validator.ValidationErrors` 이면 400 `VALIDATION_ERROR` 와 `fields`
- JSON 문법 오류나 빈 본문은 400 `BAD_REQUEST`

`internal/httpapi/posts.go` 의 `createPost`는 키가 80자를 넘으면 400이고, 10분 안에 같은 사용자와 같은 키면 저장해 둔 상태 코드와 본문, `Location`을 다시 보낸다.

화면 `project/web/app.js` 의 `withButton`은 요청 시작에 `button.disabled = true`, `finally`에서 다시 켠다. 실패하면 `error.status`와 `error.message`를 상태 줄에 쓴다. 200이면 목록을 다시 그리고, 400이면 입력 칸을 고치게 둔다. 201이면 등록 완료로 본다.

회원가입 검증은 `signupIn` 이다. 이메일은 `email`, 비밀번호는 8자 이상 72자 이하(bcrypt가 보는 최대), 별명은 1~40자다.

## ④ 🔗 연동 포인트

검증을 통과한 `title`, `body`, `tags`가 study6의 `INSERT` 입력이 된다. 화면은 `POST /api/posts` 의 상태 코드로 갈라진다.

| 상태 | 화면 |
| --- | --- |
| 201 | 목록을 다시 읽고 상세를 연다 |
| 400 | `error.message`와 `fields`를 보여 주고 버튼만 다시 켠다 |
| 401 | 로그인으로 돌린다 |
| 413 | 본문이 너무 크다고 말한다 |

스펙의 422를 쓰는 다른 서버를 붙일 일이 생기면 400과 같은 분기로 넣는다. 이 Gin 서버는 검증 실패를 400으로 통일한다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 빈 제목인데 201 | `binding` 태그가 없거나 `ShouldBind`를 안 봄 | 태그와 오류 분기를 둔다 |
| 클라이언트가 멈춘 것처럼 보임 | 버튼을 끄고 실패 경로에서 안 켬 | `finally`에서 `disabled = false` |
| 클릭 두 번에 글이 둘 | 서버가 키를 기억하지 않음 | `Idempotency-Key` 저장 |
| 데드락 | `Lock` 뒤 `Unlock` 없이 return | return 전에 잠금을 푼다 |
| CORS 오류 | 허용 출처가 `*` 이거나 화면 출처와 다름 | `CORS_ORIGIN=http://localhost:5173` |
| 비밀번호 72바이트 초과가 잘림 | bcrypt 한계 | 검증 `max=72` 로 미리 거절 |

## ⑥ 직접 해보기

1. 제목 없이 `POST /api/posts` 를 보내 400과 `VALIDATION_ERROR`를 확인한다.
2. 같은 `Idempotency-Key`로 두 번 보내 두 번째 헤더 `Idempotency-Replayed` 를 본다.
3. 화면에서 등록 버튼을 연타해도 요청이 겹치지 않는지 버튼을 보고 확인한다.

## ⑦ ☐ 체크리스트

- [ ] JSON 바인딩·검증, 미들웨어(로거, CORS, 복구)
- [ ] `binding` 태그로 필수와 길이를 검사했다
- [ ] 검증 실패는 400 `VALIDATION_ERROR` 와 `fields` 다
- [ ] JSON 문법 오류는 400 `BAD_REQUEST` 다
- [ ] 화면이 200, 201, 400에 따라 문구를 바꾼다
- [ ] 제출 중 버튼을 끄고, 끝나면 켠다
- [ ] 같은 Idempotency-Key 는 글을 한 번만 만든다
- [ ] `go test ./demo/step05` 가 통과한다

## ⑧ 다음 단계

검사된 입력을 study6에서 SQLite에 넣고 목록, 상세, 수정, 삭제로 잇는다.
