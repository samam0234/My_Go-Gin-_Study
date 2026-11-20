# study11. DB 확장

## ① 목표

같은 저장소 인터페이스로 SQLite, PostgreSQL, MariaDB를 바꾸고, GORM과 MongoDB는 그 차이에 맞는 작은 프로그램으로 확인한다.

## ② 📖 이론

게시판 스펙의 데이터는 사용자, 글, 댓글처럼 관계가 있다. SQL 데이터베이스는 이 관계에 외래 키를 둔다. MongoDB는 문서를 통째로 넣으므로 댓글 수나 작성자 닉네임을 어떻게 넣을지 스키마를 다시 정해야 한다. 그래서 본 서버의 `DB_DRIVER`는 `sqlite`, `postgres`, `mariadb`(또는 `mysql`)만 받고, MongoDB는 `demo/step11mongo` 에서 연결 방식만 연습한다.

sqlx는 SQL을 우리가 적고 결과만 구조체에 담는다. GORM은 구조체 태그로 테이블을 만들고 `Create`, `First` 를 호출한다. SQL이 짧아지지만 어떤 문장이 나가는지 바로 보이지 않는다. 쿼리를 통제할 때는 sqlx, 작은 메모를 빨리 만들 때는 GORM이 읽기 쉽다.

드라이버 차이:

| 환경 값 | Go 드라이버 | 자리 표시 | 자동 증가 |
| --- | --- | --- | --- |
| sqlite | modernc.org/sqlite | `?` | INTEGER PRIMARY KEY AUTOINCREMENT |
| postgres | pgx stdlib 이름 `pgx` | `$1` 로 Rebind | IDENTITY, `RETURNING id` |
| mariadb, mysql | go-sql-driver/mysql | `?` | AUTO_INCREMENT, LastInsertId |

`insertID`가 postgres일 때만 `RETURNING id` 를 붙인다. 검색의 `LOWER` 와 `ESCAPE` 는 세 SQL에서 같은 형태로 둔다. MySQL 테이블은 `utf8mb4` 를 붙인다.

비유하면 SQL 세 종은 같은 장부의 다른 제본이고, MongoDB는 장부 대신 서류철이다. 창구(핸들러)는 장부 양식이 같을 때만 그대로 둔다.

## ③ 🛠️ 실기

SQLite는 기본값이다.

```text
DB_DRIVER=sqlite
DB_DSN=file:board.db?_pragma=busy_timeout(5000)
```

PostgreSQL은 Compose 프로필로 DB만 띄운다.

```powershell
docker compose -f project/docker-compose.yml --profile postgres up -d postgres
```

```text
DB_DRIVER=postgres
DB_DSN=postgres://board:board@localhost:5432/board?sslmode=disable
```

MariaDB:

```powershell
docker compose -f project/docker-compose.yml --profile mariadb up -d mariadb
```

```text
DB_DRIVER=mariadb
DB_DSN=board:board@tcp(127.0.0.1:3306)/board
```

`store.normalize`가 `charset=utf8mb4` 를 DSN에 보탠다. 서버를 다시 실행하면 `Migrate`가 그 방언의 `CREATE TABLE`을 실행한다. 핸들러 코드는 그대로다.

GORM 메모는 `demo/step11gorm/main.go` 다.

```go
type Note struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func run(path string) error {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&Note{}); err != nil {
		return err
	}
	note := Note{Title: "GORM 메모", Body: "태그가 컬럼이 됩니다"}
	if err := db.Create(&note).Error; err != nil {
		return err
	}
	var found Note
	return db.First(&found, note.ID).Error
}
```

1. 태그가 기본 키와 JSON 이름을 같이 말한다. sqlx 구조체의 `db` 태그와 역할이 비슷하지만 마이그레이션까지 태그가 맡는다.
2. `gorm.Open`은 순수 Go SQLite 드라이버를 연다.
3. 테스트가 임시 파일을 지울 수 있게 `db.DB().Close()` 를 미룬다. Windows에서 이 닫기가 없으면 파일이 잠긴다.
4. `AutoMigrate`가 `notes` 테이블을 만든다. 우리가 적은 `CREATE TABLE`이 없다.
5. `Create`와 `First`가 insert와 select를 숨긴다.

```powershell
go test ./demo/step11gorm
```

MongoDB는 문서 하나다.

```go
client, err := mongo.Connect(options.Client().ApplyURI(uri))
coll := client.Database("gin_study").Collection("notes")
_, err = coll.InsertOne(ctx, note{Title: "mongo"})
err = coll.FindOne(ctx, bson.D{{Key: "title", Value: "mongo"}}).Decode(&found)
```

`MONGO_URL`이 없으면 연결하지 않고 끝난다. 드라이버 v2의 `Connect`는 컨텍스트를 첫 인자로 받지 않는다. 조회와 종료는 컨텍스트를 받는다.

```powershell
go test ./demo/step11mongo
```

본 서버에 `DB_DRIVER=mongo` 를 넣으면 시작 오류로 "sqlite, postgres, mariadb, mysql"만 안내한다. 관계형 스펙을 문서 DB에 억지로 끼우지 않기 위해서다.

## ④ 🔗 연동 포인트

환경 변수만 바꾸면 study6의 CRUD와 study7의 사용자가 다른 SQL 서버를 본다. 화면과 경로는 그대로다. GORM 예제와 Mongo 예제는 본 서버 테이블을 대체하지 않는다. studyFinal에서 배포용 Compose를 PostgreSQL로 바꿀 때도 `DB_DSN`과 볼륨만 교체한다.

## ⑤ 오류 노트

| 증상 | 원인 | 해결 |
| --- | --- | --- |
| postgres `relation does not exist` | Migrate 전에 요청 | 기동 로그의 migrate 오류를 먼저 본다 |
| postgres에서 id가 항상 0 | `RETURNING` 없이 LastInsertId | `insertID`의 postgres 분기 |
| MariaDB 한글이 깨짐 | latin1 | `utf8mb4` |
| `UNIQUE` 인데 409가 아님 | 드라이버 오류 문자열을 못 알아봄 | `unique` 또는 `duplicate` 포함 여부로 `ErrConflict` |
| GORM 테스트가 파일을 못 지움 | 연결을 안 닫음 | `sqlDB.Close()` |
| Mongo를 본 서버 DSN에 넣음 | 관계 모델과 다름 | `demo/step11mongo`로 따로 연습 |

## ⑥ 직접 해보기

1. SQLite로 글을 하나 만들고, DSN만 임시 파일로 바꿔 목록이 비는지 본다.
2. `go test ./demo/step11gorm` 으로 태그 기반 마이그레이션을 확인한다.
3. PostgreSQL을 띄울 수 있으면 `DB_DRIVER=postgres` 로 `/health` 가 200인지 본다. 없으면 DSN 형식만 노트에 적고 SQLite로 남긴다.

## ⑦ ☐ 체크리스트

- [ ] SQLite, PostgreSQL, MariaDB의 DSN을 구분했다
- [ ] SQL은 `?`로 적고 postgres는 Rebind와 RETURNING을 쓴다
- [ ] 핸들러는 드라이버가 바뀌어도 그대로다
- [ ] GORM `AutoMigrate`와 sqlx `CREATE TABLE`의 차이를 말했다
- [ ] MongoDB는 별도 예제이고 본 서버 드라이버가 아니다
- [ ] `go test ./demo/step11gorm ./demo/step11mongo` 가 통과한다

## ⑧ 다음 단계

DB가 스펙 JSON을 만들면 study12에서 그 JSON을 브라우저 화면에 글자로 붙인다.
