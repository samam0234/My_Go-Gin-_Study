# study6. DB와 API 연동

## ① 목표

SQLite를 `database/sql`과 sqlx로 연결하고, 공통 스펙 v1의 건강 체크와 글 CRUD를 환경 변수에 있는 접속 정보로 동작하게 한다.

## ② 📖 이론

`database/sql`은 Go 표준 데이터베이스 인터페이스다. 드라이버를 import 하면 `sql.Open`이 그 이름을 안다. sqlx는 그 위에 구조체 스캔(`Get`, `Select`)을 더한다. GORM은 구조체 태그로 테이블을 만드는 다른 선택이다. 이 서버의 본 경로는 sqlx이고, GORM은 study11의 `demo/step11gorm` 에서 같은 메모를 다시 만든다.

연결 문자열은 코드에 적지 않는다. `DB_DRIVER`와 `DB_DSN`이다. 기본값은 SQLite 파일이다. C 컴파일러 없이 Windows에서 빌드하려고 `modernc.org/sqlite` 를 쓴다. 드라이버 이름은 `sqlite` 다.

한 줄의 안전 규칙은 이렇다. 사용자가 보낸 글자는 SQL 문장에 붙이지 않고 `?` 자리에 넘긴다. 정렬처럼 문장 조각이 꼭 필요하면 `createdAt`, `-createdAt`, `-commentCount` 만 골라 우리가 쓴 문자열을 넣는다.

비유하면 DB는 장부이고, 드라이버는 장부 프로그램이며, sqlx는 장부 칸을 구조체 필드에 옮겨 주는 자 이다. DSN은 장부가 있는 서랍 번호라서 서랍을 바꾸면 환경 변수만 바꾼다.

## ③ 🛠️ 실기

작은 확인은 `demo/step06` 다.

```powershell
go test ./demo/step06
```

```go
func run(path string) error {
	st, err := store.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		return err
	}
	defer st.Close()
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		return err
	}
	user, err := st.CreateUser(ctx, "mina@example.com", "hash-for-demo-only", "mina")
	if err == store.ErrConflict {
		user, err = st.FindUserByEmail(ctx, "mina@example.com")
	}
	if err != nil {
		return err
	}
	if _, err := st.CreatePost(ctx, user.ID, "첫 글", "안녕하세요", []string{"intro"}); err != nil {
		return err
	}
	_, _, err = st.ListPosts(ctx, store.PostFilter{Page: 1, Limit: 10, Sort: "-createdAt"})
	return err
}
```

1. 경로는 `file:` 접두어와 슬래시로 맞춘다. Windows 역슬래시는 SQLite DSN에서 깨지기 쉽다.
2. `Open`이 드라이버를 고르고 `Ping` 한다. SQLite는 연결을 1개로 제한하고 `PRAGMA foreign_keys = ON` 을 켠다.
3. `defer Close`로 함수가 어떻게 끝나도 파일을 닫는다.
4. `Migrate`는 `CREATE TABLE IF NOT EXISTS` 를 순서대로 실행한다. 사용자, 글, 댓글, 문서, 질문, 사용량, 메일, 결제가 있다.
5. 사용자를 만든다. 이메일은 소문자로 저장한다.
6. 이미 있으면 `ErrConflict` 이므로 찾아서 계속한다. 이 단계의 해시는 예시 문자열이다. 진짜 비밀번호 해시는 study7이다.
7. 글을 만들고 목록을 읽는다. 목록은 `(행, 전체 개수, 에러)` 다.

스펙에 맞춘 HTTP는 `internal/httpapi/posts.go` 와 `internal/store/store.go` 다.

| 메서드 | 경로 | 성공 | 설명 |
| --- | --- | --- | --- |
| GET | `/health` | 200 | `{"status":"ok"}`. DB ping 실패면 500 |
| GET | `/api/posts` | 200 | `data` 배열과 `meta` |
| GET | `/api/posts/{id}` | 200 | 본문, 태그, `updatedAt` 포함 |
| POST | `/api/posts` | 201 | `Location: /api/posts/{id}` |
| PATCH | `/api/posts/{id}` | 200 | 보낸 필드만 바꾼다 |
| DELETE | `/api/posts/{id}` | 204 | 본문 없음 |

목록 한 항목은 `id`, `title`, `author`, `commentCount`, `createdAt` 만 있다. 상세와 등록 응답에는 `body`, `tags`, `updatedAt` 이 더 있다. 작성자 객체는 `id`와 `nickname` 만 가진다.

목록 쿼리:

| 이름 | 기본 | 범위 |
| --- | --- | --- |
| page | 1 | 1 이상 |
| limit | 10 | 1~50, 넘치면 50 |
| q | 없음 | 제목·본문 부분 일치 |
| sort | `-createdAt` | `createdAt`, `-createdAt`, `-commentCount` |

`meta`는 `page`, `limit`, `total`, `totalPages`, `q` 다. 검색어의 `%` 와 `_` 는 LIKE 패턴이 되지 않게 이스케이프한다.

댓글은 `GET/POST /api/posts/{id}/comments`, `DELETE /api/comments/{id}` 다. 글이 없으면 댓글 목록도 404다. 글을 지우면 SQLite 외래 키가 댓글을 함께 지운다.

없는 행은 `sql.ErrNoRows`를 `store.ErrNotFound`로 바꾸고, 핸들러가 404 `NOT_FOUND`로 응답한다. 그 외 DB 오류는 로그에만 남기고 500 `INTERNAL` 이다. 응답에 SQL 문장을 넣지 않는다.

환경 변수 예:

```text
DB_DRIVER=sqlite
DB_DSN=file:board.db?_pragma=busy_timeout(5000)
```

`config.Load`가 `.env`를 읽고, `store.Open(cfg.Driver, cfg.DSN)` 이 연결한다. 서버 시작 시 `Migrate`가 테이블을 만든다. `SEED=1`이면 `mina@example.com` / `password123` 과 제목 `첫 글` 을 넣는다.

## ④ 🔗 연동 포인트

이 CRUD가 공통 스펙 v1의 글 구간이다. 화면 `project/web/app.js` 가 같은 경로를 호출한다. 등록은 201과 `data.id`로 상세를 연다. 접속 정보를 환경 변수로 빼 두었으므로 study11에서 `DB_DRIVER=postgres` 또는 `mariadb` 로 바꿀 수 있다. 인증이 붙기 전에는 수정과 삭제가 401이다. study7이 Bearer 토큰을 만든다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| `no such table` | Migrate 전에 조회 | 시작 시 `Migrate` |
| 댓글을 지운 뒤에도 남음 | 외래 키 프래그마가 꺼짐 | `PRAGMA foreign_keys = ON` 과 연결 1개 |
| `database is locked` | 여러 연결이 파일을 씀 | `SetMaxOpenConns(1)` |
| 검색에 `%`를 넣으면 전부 나옴 | LIKE 와일드카드 | `escapeLike` |
| 목록이 `null` | nil 슬라이스 | 조회 결과가 nil이면 빈 슬라이스로 바꾼다 |
| 응답에 SQL 오류가 그대로 보임 | `err.Error()`를 JSON에 넣음 | 로그에만 남기고 `INTERNAL` |
| Windows에서 DSN 실패 | 역슬래시 경로 | `filepath.ToSlash` |

## ⑥ 직접 해보기

1. `SEED=1`로 서버를 켜고 `GET /api/posts?q=안녕` 의 `meta.total` 을 본다.
2. 글을 지운 뒤 그 글의 댓글 삭제가 404인지 확인한다.
3. `DB_DSN`을 다른 파일로 바꿔 빈 목록이 되는지 본다. 끝나면 원래 값으로 되돌린다.

## ⑦ ☐ 체크리스트

- [ ] DB: database/sql + sqlx 또는 GORM
- [ ] 연결 정보가 `DB_DRIVER`, `DB_DSN` 에 있다
- [ ] 마이그레이션으로 테이블을 만든다
- [ ] 목록, 상세, 등록, 수정, 삭제가 스펙의 상태 코드와 같다
- [ ] 목록 응답은 `data`와 `meta` 다
- [ ] 등록은 201과 `Location` 이다
- [ ] 삭제는 204이고 본문이 없다
- [ ] SQL 자리는 `?` 이고 정렬은 허용 목록이다
- [ ] `go test ./internal/httpapi` 의 글 시나리오가 통과한다

## ⑧ 다음 단계

장부에 사용자를 넣었다. study7에서 비밀번호를 해시하고, 그 사용자만 글을 고치게 한다.
