# study3. 데이터 다루기

## ① 목표

구조체와 JSON 태그로 값을 표현하고, 그 값을 파일에 저장한 뒤 다시 읽어 같은 구조체로 되돌린다.

## ② 📖 이론

웹 API의 데이터는 대부분 구조체다. 필드는 메모리 안의 자리이고, `json` 태그는 그 자리가 선으로 나갈 때의 이름이다. 공통 스펙은 `createdAt` 처럼 camelCase를 쓴다. Go 필드가 `CreatedAt` 이어도 태그가 `` `json:"createdAt"` `` 이면 맞출 수 있다.

슬라이스는 길이가 변하는 목록이다. `append`로 항목을 더한다. 맵은 키로 찾는 상자다. 없는 키를 읽으면 제로 값과 함께 `ok == false` 가 나온다.

```go
names := []string{"kim", "lee"}
names = append(names, "park")
scores := map[string]int{"kim": 90}
value, ok := scores["choi"]
```

파일을 저장소로 쓰면 DB 없이 목록이 프로세스를 꺼도 남는다. 한 명이 쓸 때는 간단하지만, 두 요청이 동시에 같은 파일을 고치면 한쪽이 잃는다. 그래서 학습용 메모는 파일이고, 게시판은 study6에서 SQLite로 옮긴다.

화면과 묶는 바인딩은 다음 두 방향이다.

- 나갈 때: 구조체 → `json.Marshal` → 파일 또는 `c.JSON`
- 들어올 때: JSON 바이트 → `json.Unmarshal` → 구조체 포인터

비유하면 구조체는 서식이고, JSON은 그 서식을 봉투에 적은 것이다. 태그 이름이 봉투의 칸 이름이다.

## ③ 🛠️ 실기

`demo/step03/main.go` 의 `run`이 한 사이클이다.

```go
type Memo struct {
	Text string `json:"text"`
}

func run(path string) error {
	memos := []Memo{{Text: "파일에 남긴 메모"}}
	raw, err := json.MarshalIndent(memos, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return err
	}
	loaded, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var back []Memo
	return json.Unmarshal(loaded, &back)
}
```

1. `Memo`는 텍스트 하나다. JSON 키는 `text` 다.
2. 슬라이스에 메모 한 건을 넣는다.
3. 들여쓰기 있는 JSON 바이트로 바꾼다. 사람이 파일을 열어 볼 수 있다.
4. 변환 실패는 바로 반환한다. 파일을 쓰지 않는다.
5. 권한 `0644`로 저장한다. 소유자는 읽고 쓰고, 다른 사용자는 읽기만 한다.
6. 저장 실패도 반환한다.
7. 같은 경로를 다시 읽는다.
8. 읽기 실패를 반환한다.
9. `back`은 빈 슬라이스 변수다. `Unmarshal`에는 포인터 `&back` 을 넘긴다. 포인터가 아니면 함수 안의 복사본만 채워지고 호출자는 빈 값이 된다.
10. 테스트는 임시 폴더 경로를 넘겨 저장소에 `memos.json`을 남기지 않는다.

게시판 서버의 글 구조체는 `internal/store/store.go` 의 `Post` 다. DB 컬럼은 `db` 태그, 화면 JSON은 핸들러가 `postDetailJSON`으로 만든다. 비밀번호 해시는 `User.PasswordHash`에만 있고 응답 함수 `publicUser`는 `id`, `email`, `nickname`만 고른다. 구조체에 있다고 모두 나가지 않는다.

로컬 화면 `project/web/app.js` 는 `fetch`로 받은 `data.title` 을 `textContent`에 넣는다. 이것이 브라우저 쪽 바인딩이다. HTML 문자열로 끼워 넣지 않는다.

## ④ 🔗 연동 포인트

파일에 남긴 `[]Memo` 는 study6 `posts` 테이블의 한 행과 같은 역할이다. JSON 키 `text`를 스펙의 `title`, `body`, `tags`로 늘리면 등록 본문이 된다. 스펙은 시각을 `2026-11-01T09:00:00Z` 같은 UTC 문자열로 둔다. 서버는 `time.RFC3339`로 저장한다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| 다시 읽은 슬라이스가 비어 있음 | `Unmarshal`에 포인터를 안 넘김 | `&back` |
| JSON이 `Text`로 나감 | 태그 없음 | `json:"text"` |
| `unexpected end of JSON` | 파일을 쓰다 말았거나 빈 파일 | 쓰기 오류를 무시하지 않는다 |
| 동시에 두 요청이 파일을 덮어씀 | 잠금 없는 파일 저장 | 게시판 데이터는 DB로 옮긴다 |
| 응답에 비밀번호 해시가 보임 | 엔티티를 그대로 `c.JSON` | 공개용 맵만 만든다 |

## ⑥ 직접 해보기

1. `Memo`에 `CreatedAt string \`json:"createdAt"\`` 를 넣고 파일에 키가 camelCase로 생기는지 연다.
2. 일부러 `&back` 에서 `&`를 빼 보고 컴파일 또는 빈 결과를 확인한 뒤 되돌린다.
3. `go test ./demo/step03` 으로 임시 파일 왕복을 확인한다.

## ⑦ ☐ 체크리스트

- [ ] 구조체 필드와 JSON 키를 태그로 구분했다
- [ ] 슬라이스에 값을 넣고 `Marshal` 했다
- [ ] 파일을 쓰고 다시 `Unmarshal` 했다
- [ ] 에러가 있으면 다음 단계로 가지 않았다
- [ ] 응답에 넣을 필드와 저장만 할 필드를 나눴다
- [ ] 화면에는 `textContent`로 글자를 넣는다

## ⑧ 다음 단계

파일로 남긴 목록을 study4에서 URL 경로와 쿼리로 고르게 한다.
