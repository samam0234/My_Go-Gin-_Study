package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type Store struct {
	db      *sqlx.DB
	logical string
}

type User struct {
	ID           int64  `db:"id"`
	Email        string `db:"email"`
	PasswordHash string `db:"password_hash"`
	Nickname     string `db:"nickname"`
	Role         string `db:"role"`
	CreatedAt    string `db:"created_at"`
}

type Post struct {
	ID             int64  `db:"id"`
	AuthorID       int64  `db:"author_id"`
	AuthorNickname string `db:"author_nickname"`
	Title          string `db:"title"`
	Body           string `db:"body"`
	Tags           string `db:"tags"`
	CommentCount   int    `db:"comment_count"`
	CreatedAt      string `db:"created_at"`
	UpdatedAt      string `db:"updated_at"`
}

type Comment struct {
	ID             int64  `db:"id"`
	PostID         int64  `db:"post_id"`
	AuthorID       int64  `db:"author_id"`
	AuthorNickname string `db:"author_nickname"`
	Body           string `db:"body"`
	CreatedAt      string `db:"created_at"`
}

type Document struct {
	ID        int64  `db:"id"`
	OwnerID   int64  `db:"owner_id"`
	Title     string `db:"title"`
	Content   string `db:"content"`
	Summary   string `db:"summary"`
	CreatedAt string `db:"created_at"`
}

type Question struct {
	ID         int64  `db:"id"`
	DocumentID int64  `db:"document_id"`
	Question   string `db:"question"`
	Answer     string `db:"answer"`
	Sources    string `db:"sources"`
	CreatedAt  string `db:"created_at"`
}

type Payment struct {
	ID        int64  `db:"id"`
	UserID    int64  `db:"user_id"`
	OrderID   string `db:"order_id"`
	Amount    int64  `db:"amount"`
	Status    string `db:"status"`
	CreatedAt string `db:"created_at"`
}

type Mail struct {
	ID        int64  `db:"id"`
	ToEmail   string `db:"to_email"`
	Subject   string `db:"subject"`
	Body      string `db:"body"`
	Status    string `db:"status"`
	CreatedAt string `db:"created_at"`
}

type PostFilter struct {
	Q     string
	Page  int
	Limit int
	Sort  string
}

func Open(driver, dsn string) (*Store, error) {
	logical, sqlDriver, outDSN, err := normalize(driver, dsn)
	if err != nil {
		return nil, err
	}
	db, err := sqlx.Open(sqlDriver, outDSN)
	if err != nil {
		return nil, err
	}
	if logical == "sqlite" {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(10)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if logical == "sqlite" {
		if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return &Store{db: db, logical: logical}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) Logical() string { return s.logical }

func (s *Store) Migrate(ctx context.Context) error {
	for i, stmt := range ddl(s.logical) {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate %s step %d: %w", s.logical, i, err)
		}
	}
	return nil
}

func (s *Store) CreateUser(ctx context.Context, email, hash, nickname string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	id, err := s.insertID(ctx, `INSERT INTO users (email, password_hash, nickname, role, created_at) VALUES (?, ?, ?, 'user', ?)`,
		email, hash, strings.TrimSpace(nickname), now())
	if IsConflict(err) {
		return User{}, ErrConflict
	}
	if err != nil {
		return User{}, err
	}
	return s.FindUserByID(ctx, id)
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := s.db.GetContext(ctx, &user, s.q(`SELECT id, email, password_hash, nickname, role, created_at FROM users WHERE email = ?`), strings.ToLower(strings.TrimSpace(email)))
	return one(user, err)
}

func (s *Store) FindUserByID(ctx context.Context, id int64) (User, error) {
	var user User
	err := s.db.GetContext(ctx, &user, s.q(`SELECT id, email, password_hash, nickname, role, created_at FROM users WHERE id = ?`), id)
	return one(user, err)
}

func (s *Store) SetRole(ctx context.Context, id int64, role string) error {
	res, err := s.db.ExecContext(ctx, s.q(`UPDATE users SET role = ? WHERE id = ?`), role, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreatePost(ctx context.Context, authorID int64, title, body string, tags []string) (Post, error) {
	raw, err := json.Marshal(nonNil(tags))
	if err != nil {
		return Post{}, err
	}
	stamp := now()
	id, err := s.insertID(ctx, `INSERT INTO posts (author_id, title, body, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		authorID, title, body, string(raw), stamp, stamp)
	if err != nil {
		return Post{}, err
	}
	return s.GetPost(ctx, id)
}

func (s *Store) GetPost(ctx context.Context, id int64) (Post, error) {
	var post Post
	err := s.db.GetContext(ctx, &post, s.q(postSelect+` WHERE p.id = ?`), id)
	return one(post, err)
}

func (s *Store) ListPosts(ctx context.Context, f PostFilter) ([]Post, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 50 {
		f.Limit = 10
	}
	order, err := orderClause(f.Sort)
	if err != nil {
		return nil, 0, err
	}
	where := "WHERE 1=1"
	args := []any{}
	if q := strings.TrimSpace(f.Q); q != "" {
		like := "%" + escapeLike(strings.ToLower(q)) + "%"
		where += " AND (LOWER(p.title) LIKE ? ESCAPE '\\' OR LOWER(p.body) LIKE ? ESCAPE '\\')"
		args = append(args, like, like)
	}
	var total int
	if err := s.db.GetContext(ctx, &total, s.q(`SELECT COUNT(*) FROM posts p `+where), args...); err != nil {
		return nil, 0, err
	}
	query := postSelect + " " + where + " ORDER BY " + order + " LIMIT ? OFFSET ?"
	listArgs := append(append([]any{}, args...), f.Limit, (f.Page-1)*f.Limit)
	var rows []Post
	if err := s.db.SelectContext(ctx, &rows, s.q(query), listArgs...); err != nil {
		return nil, 0, err
	}
	if rows == nil {
		rows = []Post{}
	}
	return rows, total, nil
}

func (s *Store) UpdatePost(ctx context.Context, id int64, title, body string, tags []string) (Post, error) {
	raw, err := json.Marshal(nonNil(tags))
	if err != nil {
		return Post{}, err
	}
	res, err := s.db.ExecContext(ctx, s.q(`UPDATE posts SET title = ?, body = ?, tags = ?, updated_at = ? WHERE id = ?`),
		title, body, string(raw), now(), id)
	if err != nil {
		return Post{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Post{}, ErrNotFound
	}
	return s.GetPost(ctx, id)
}

func (s *Store) DeletePost(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, s.q(`DELETE FROM posts WHERE id = ?`), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CountPosts(ctx context.Context) (int, error) {
	var n int
	err := s.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM posts`)
	return n, err
}

func (s *Store) CreateComment(ctx context.Context, postID, authorID int64, body string) (Comment, error) {
	id, err := s.insertID(ctx, `INSERT INTO comments (post_id, author_id, body, created_at) VALUES (?, ?, ?, ?)`,
		postID, authorID, body, now())
	if err != nil {
		return Comment{}, err
	}
	return s.GetComment(ctx, id)
}

func (s *Store) ListComments(ctx context.Context, postID int64) ([]Comment, error) {
	var rows []Comment
	err := s.db.SelectContext(ctx, &rows, s.q(commentSelect+` WHERE c.post_id = ? ORDER BY c.id ASC`), postID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []Comment{}
	}
	return rows, nil
}

func (s *Store) GetComment(ctx context.Context, id int64) (Comment, error) {
	var row Comment
	err := s.db.GetContext(ctx, &row, s.q(commentSelect+` WHERE c.id = ?`), id)
	return one(row, err)
}

func (s *Store) DeleteComment(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, s.q(`DELETE FROM comments WHERE id = ?`), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateDocument(ctx context.Context, ownerID int64, title, content, summary string) (Document, error) {
	id, err := s.insertID(ctx, `INSERT INTO documents (owner_id, title, content, summary, created_at) VALUES (?, ?, ?, ?, ?)`,
		ownerID, title, content, summary, now())
	if err != nil {
		return Document{}, err
	}
	return s.GetDocument(ctx, id)
}

func (s *Store) ListDocuments(ctx context.Context, ownerID int64) ([]Document, error) {
	var rows []Document
	err := s.db.SelectContext(ctx, &rows, s.q(`SELECT id, owner_id, title, '' AS content, summary, created_at FROM documents WHERE owner_id = ? ORDER BY id DESC`), ownerID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []Document{}
	}
	return rows, nil
}

func (s *Store) GetDocument(ctx context.Context, id int64) (Document, error) {
	var row Document
	err := s.db.GetContext(ctx, &row, s.q(`SELECT id, owner_id, title, content, summary, created_at FROM documents WHERE id = ?`), id)
	return one(row, err)
}

func (s *Store) CreateQuestion(ctx context.Context, documentID int64, question, answer, sources string) (Question, error) {
	id, err := s.insertID(ctx, `INSERT INTO questions (document_id, question, answer, sources, created_at) VALUES (?, ?, ?, ?, ?)`,
		documentID, question, answer, sources, now())
	if err != nil {
		return Question{}, err
	}
	var row Question
	err = s.db.GetContext(ctx, &row, s.q(`SELECT id, document_id, question, answer, sources, created_at FROM questions WHERE id = ?`), id)
	return one(row, err)
}

func (s *Store) ListQuestions(ctx context.Context, documentID int64) ([]Question, error) {
	var rows []Question
	err := s.db.SelectContext(ctx, &rows, s.q(`SELECT id, document_id, question, answer, sources, created_at FROM questions WHERE document_id = ? ORDER BY id ASC`), documentID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []Question{}
	}
	return rows, nil
}

func (s *Store) AddUsage(ctx context.Context, userID int64, inputTokens, outputTokens int) error {
	_, err := s.insertID(ctx, `INSERT INTO ai_usage (user_id, input_tokens, output_tokens, created_at) VALUES (?, ?, ?, ?)`,
		userID, inputTokens, outputTokens, now())
	return err
}

func (s *Store) UsageToday(ctx context.Context, userID int64, now time.Time) (int, error) {
	start := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	var total int
	err := s.db.GetContext(ctx, &total, s.q(`SELECT COALESCE(SUM(input_tokens + output_tokens), 0) FROM ai_usage WHERE user_id = ? AND created_at >= ?`), userID, start)
	return total, err
}

func (s *Store) EnqueueMail(ctx context.Context, to, subject, body string) error {
	_, err := s.insertID(ctx, `INSERT INTO mail_outbox (to_email, subject, body, status, created_at) VALUES (?, ?, ?, 'pending', ?)`,
		to, subject, body, now())
	return err
}

func (s *Store) NextPendingMail(ctx context.Context) (Mail, error) {
	var row Mail
	err := s.db.GetContext(ctx, &row, s.q(`SELECT id, to_email, subject, body, status, created_at FROM mail_outbox WHERE status = 'pending' ORDER BY id LIMIT 1`))
	return one(row, err)
}

func (s *Store) MarkMailSent(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE mail_outbox SET status = 'sent' WHERE id = ?`), id)
	return err
}

func (s *Store) CreatePayment(ctx context.Context, userID int64, orderID string, amount int64) (Payment, error) {
	id, err := s.insertID(ctx, `INSERT INTO payments (user_id, order_id, amount, status, created_at) VALUES (?, ?, ?, 'pending', ?)`,
		userID, orderID, amount, now())
	if IsConflict(err) {
		return Payment{}, ErrConflict
	}
	if err != nil {
		return Payment{}, err
	}
	return s.GetPayment(ctx, id)
}

func (s *Store) FindPaymentByOrder(ctx context.Context, userID int64, orderID string) (Payment, error) {
	var row Payment
	err := s.db.GetContext(ctx, &row, s.q(`SELECT id, user_id, order_id, amount, status, created_at FROM payments WHERE user_id = ? AND order_id = ?`), userID, orderID)
	return one(row, err)
}

func (s *Store) GetPayment(ctx context.Context, id int64) (Payment, error) {
	var row Payment
	err := s.db.GetContext(ctx, &row, s.q(`SELECT id, user_id, order_id, amount, status, created_at FROM payments WHERE id = ?`), id)
	return one(row, err)
}

func (s *Store) ConfirmPayment(ctx context.Context, id, userID int64) (Payment, error) {
	res, err := s.db.ExecContext(ctx, s.q(`UPDATE payments SET status = 'paid' WHERE id = ? AND user_id = ?`), id, userID)
	if err != nil {
		return Payment{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Payment{}, ErrNotFound
	}
	return s.GetPayment(ctx, id)
}

func (s *Store) insertID(ctx context.Context, query string, args ...any) (int64, error) {
	q := s.db.Rebind(query)
	if s.logical == "postgres" {
		var id int64
		err := s.db.QueryRowContext(ctx, q+" RETURNING id", args...).Scan(&id)
		return id, err
	}
	res, err := s.db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) q(query string) string { return s.db.Rebind(query) }

func one[T any](row T, err error) (T, error) {
	if errors.Is(err, sql.ErrNoRows) {
		var zero T
		return zero, ErrNotFound
	}
	return row, err
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func nonNil(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

func IsConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func orderClause(sort string) (string, error) {
	switch sort {
	case "", "-createdAt":
		return "p.created_at DESC", nil
	case "createdAt":
		return "p.created_at ASC", nil
	case "-commentCount":
		return "comment_count DESC, p.created_at DESC", nil
	default:
		return "", fmt.Errorf("sort %q", sort)
	}
}

const postSelect = `
SELECT p.id, p.author_id, u.nickname AS author_nickname, p.title, p.body, p.tags,
       p.created_at, p.updated_at,
       (SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id) AS comment_count
FROM posts p
JOIN users u ON u.id = p.author_id`

const commentSelect = `
SELECT c.id, c.post_id, c.author_id, u.nickname AS author_nickname, c.body, c.created_at
FROM comments c
JOIN users u ON u.id = c.author_id`
