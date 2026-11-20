package httpapi

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/samam0234/my-go-gin-study/internal/store"
)

func publicUser(u store.User) gin.H {
	return gin.H{"id": u.ID, "email": u.Email, "nickname": u.Nickname}
}

func authorJSON(id int64, nickname string) gin.H {
	return gin.H{"id": id, "nickname": nickname}
}

func postListJSON(p store.Post) gin.H {
	return gin.H{
		"id":           p.ID,
		"title":        p.Title,
		"author":       authorJSON(p.AuthorID, p.AuthorNickname),
		"commentCount": p.CommentCount,
		"createdAt":    p.CreatedAt,
	}
}

func postDetailJSON(p store.Post) gin.H {
	item := postListJSON(p)
	item["body"] = p.Body
	item["tags"] = decodeTags(p.Tags)
	item["updatedAt"] = p.UpdatedAt
	return item
}

func commentJSON(row store.Comment) gin.H {
	return gin.H{
		"id":        row.ID,
		"body":      row.Body,
		"author":    authorJSON(row.AuthorID, row.AuthorNickname),
		"createdAt": row.CreatedAt,
	}
}

func documentListJSON(row store.Document) gin.H {
	return gin.H{
		"id":        row.ID,
		"title":     row.Title,
		"summary":   row.Summary,
		"createdAt": row.CreatedAt,
	}
}

func documentDetailJSON(row store.Document) gin.H {
	item := documentListJSON(row)
	item["content"] = row.Content
	return item
}

func questionJSON(row store.Question) gin.H {
	var sources []any
	if err := json.Unmarshal([]byte(row.Sources), &sources); err != nil || sources == nil {
		sources = []any{}
	}
	return gin.H{
		"id":        row.ID,
		"question":  row.Question,
		"answer":    row.Answer,
		"sources":   sources,
		"createdAt": row.CreatedAt,
	}
}

func paymentJSON(row store.Payment) gin.H {
	return gin.H{
		"id":        row.ID,
		"orderId":   row.OrderID,
		"amount":    row.Amount,
		"status":    row.Status,
		"createdAt": row.CreatedAt,
	}
}

func decodeTags(raw string) []string {
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil || tags == nil {
		return []string{}
	}
	return tags
}

func canEdit(user store.User, authorID int64) bool {
	return user.ID == authorID || user.Role == "admin"
}
