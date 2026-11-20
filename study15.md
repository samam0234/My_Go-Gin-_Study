# study15. 성능과 캐시

## ① 목표

글 목록을 짧은 시간 메모리에 두고, 가입 메일은 요청을 붙잡지 않는 대기열에서 처리하며, 그 효과를 헤더와 테스트로 확인한다.

## ② 📖 이론

목록 조회는 같은 쿼리면 결과가 잠깐 같다. 매 요청마다 COUNT와 SELECT를 칠 필요가 없다. 대신 글이 생기거나 바뀌면 그 기억을 지워야 한다. 너무 오래 기억하면 방금 쓴 글이 안 보인다. 이 서버는 5초다.

캐시 키는 `posts?` 와 쿼리 문자열이다. 응답 헤더 `X-Cache` 가 `MISS` 또는 `HIT` 다. 등록, 수정, 삭제, 댓글 변경은 `DeletePrefix("posts?")` 로 목록 캐시를 비운다. 상세 조회는 캐시하지 않아 방금 수정한 본문을 바로 본다.

메일은 요청 안에서 SMTP를 기다리지 않는다. `EnqueueMail`이 행을 넣고, `RunMailer`가 2초마다 `ProcessOutbox`로 한 통을 로그에 남기고 `sent`로 바꾼다. 테스트는 루프를 돌리지 않고 `ProcessOutbox`를 한 번 호출한다.

측정은 거창한 도구 전에 헤더와 테스트로 한다.

- 같은 목록 URL을 두 번 호출해 두 번째가 HIT인지
- 글을 만든 직후 같은 URL이 MISS인지
- `go test` 시간에 bcrypt가 큰지. 가입 테스트가 느리면 해시 횟수를 시나리오당 필요한 만큼만 둔다

호출 제한 `RATE_LIMIT_PER_MIN`은 성능이라기보다 폭주를 막는 장치다. 양수일 때만 429 `RATE_LIMITED` 다.

비유하면 캐시는 5초짜리 사본이고, 대기열은 계산대 옆 발송 상자다. 사본은 원장이 바뀌면 버린다.

## ③ 🛠️ 실기

캐시 구현은 `internal/cache/cache.go` 다. `Get`은 만료면 지우고 miss다. `Set`은 본문 바이트를 복사해 둔다. 호출자가 슬라이스를 나중에 고쳐도 캐시가 바뀌지 않는다.

목록 핸들러:

```go
key := "posts?" + c.Request.URL.RawQuery
if cached, ok := s.cache.Get(key, time.Now()); ok {
	c.Header("X-Cache", "HIT")
	writeRaw(c, http.StatusOK, cached)
	return
}
```

저장 직전:

```go
s.cache.Set(key, body, time.Now(), 5*time.Second)
c.Header("X-Cache", "MISS")
```

워커는 `internal/app/run.go` 가 서버와 함께 고루틴으로 시작한다. 종료 신호를 받으면 고루틴도 `ctx.Done`으로 멈춘다.

```go
go srv.RunMailer(ctx)
```

확인:

```powershell
go test ./internal/httpapi -run TestBoardFlow -count=1
```

테스트가 목록을 두 번 읽고 `X-Cache`를 검사한다. 메일 테스트 `POST /api/mail/test` 는 202와 `status:"sent"` 를 기대한다.

느린 지점을 볼 때는 테스트에 `-count=1` 을 붙여 캐시된 테스트 결과를 쓰지 않고, 로그의 지연 시간(접근 로그 마지막 칸)을 비교한다. 개발 모드가 아니면 접근 로그가 경로와 상태만 남긴다.

## ④ 🔗 연동 포인트

캐시는 프로세스 메모리다. 서버를 두 대 띄우면 각자 5초가 어긋날 수 있다. 그때는 이 `TTL` 자리를 Redis 같은 외부 저장으로 바꾸되, 키 이름 `posts?` 와 무효화 지점은 유지한다. 메일 대기열은 이미 테이블이라 프로세스가 죽어도 행은 남는다. 다시 켜면 워커가 pending을 이어서 처리한다. study16에서 프로세스를 여러 개 두지 않는 한 SQLite와 이 메모리 캐시는 한 컨테이너에 맞다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 새 글이 목록에 5초간 없음 | 쓰기 경로가 캐시를 안 지움 | 변경 핸들러마다 `DeletePrefix` |
| HIT 본문이 다른 쿼리와 섞임 | 키에 쿼리가 빠짐 | `RawQuery`를 키에 포함 |
| 메일 테스트가 202가 아님 | 대기열이 비었는데 오류를 500으로 봄 | pending이 없으면 `ErrNotFound`는 성공으로 끝낸다 |
| 테스트가 끝나지 않음 | 테스트가 `RunMailer` 무한 루프를 켬 | 테스트는 `ProcessOutbox` 한 번 |
| 429가 개발 중에 자주 남 | 제한이 켜져 있음 | `RATE_LIMIT_PER_MIN=0` |

## ⑥ 직접 해보기

1. 같은 `/api/posts?q=안녕` 을 두 번 호출해 `X-Cache`가 MISS 다음 HIT인지 본다.
2. 글을 하나 만든 뒤 같은 주소가 MISS로 돌아오는지 본다.
3. `POST /api/mail/test` 후 터미널에 `mail to=` 가 있는지 본다.

## ⑦ ☐ 체크리스트

- [ ] 목록만 5초 캐시하고 상세는 매번 읽는다
- [ ] 쓰기 뒤에 목록 캐시를 지운다
- [ ] `X-Cache`로 적중을 본다
- [ ] 메일은 테이블 대기열과 워커로 나뉜다
- [ ] 테스트는 워커 루프를 돌리지 않는다
- [ ] 호출 제한은 환경 변수로 끄고 켤 수 있다

## ⑧ 다음 단계

캐시와 대기열이 한 프로세스 안에서 도는지 확인했으면 study16에서 그 프로세스를 이미지와 CI로 올린다.
