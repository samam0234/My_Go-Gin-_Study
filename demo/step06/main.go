package main

import (
	"context"
	"path/filepath"

	"github.com/samam0234/my-go-gin-study/internal/store"
)

func main() {
	if err := run(filepath.Join(".", "step06.db")); err != nil {
		panic(err)
	}
}

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
