package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

func writeError(c *gin.Context, status int, code, message string, fields map[string][]string) {
	errBody := gin.H{
		"code":      code,
		"message":   message,
		"requestId": c.GetString(requestIDKey),
	}
	if len(fields) > 0 {
		errBody["fields"] = fields
	}
	c.AbortWithStatusJSON(status, gin.H{"error": errBody})
}

func writeData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

func writeRaw(c *gin.Context, status int, body []byte) {
	c.Data(status, "application/json; charset=utf-8", body)
}

func dataBytes(data any) []byte {
	raw, err := json.Marshal(gin.H{"data": data})
	if err != nil {
		return []byte(`{"error":{"code":"INTERNAL","message":"응답을 만들지 못했습니다"}}`)
	}
	return raw
}

func listBytes(data any, meta gin.H) []byte {
	raw, err := json.Marshal(gin.H{"data": data, "meta": meta})
	if err != nil {
		return []byte(`{"error":{"code":"INTERNAL","message":"응답을 만들지 못했습니다"}}`)
	}
	return raw
}

func noContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
