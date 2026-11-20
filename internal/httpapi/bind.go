package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func bindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	err := c.ShouldBindJSON(dst)
	if err == nil {
		return true
	}
	if isTooLarge(err) {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "요청이 너무 큽니다.", nil)
		return false
	}
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		writeError(c, http.StatusBadRequest, "VALIDATION_ERROR", "입력값을 확인하세요.", toFields(ve))
		return false
	}
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "JSON 형식을 확인하세요.", nil)
		return false
	}
	writeError(c, http.StatusBadRequest, "BAD_REQUEST", "요청 본문을 확인하세요.", nil)
	return false
}

func isTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return true
	}
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "too large")
}

func toFields(ve validator.ValidationErrors) map[string][]string {
	out := map[string][]string{}
	for _, fe := range ve {
		name := fieldName(fe)
		out[name] = append(out[name], fieldMessage(fe))
	}
	return out
}

func fieldName(fe validator.FieldError) string {
	if strings.Contains(fe.Namespace(), "Tags") {
		return "tags"
	}
	switch fe.Field() {
	case "Email":
		return "email"
	case "Password":
		return "password"
	case "Nickname":
		return "nickname"
	case "Title":
		return "title"
	case "Body":
		return "body"
	case "Tags":
		return "tags"
	case "Messages":
		return "messages"
	case "Content":
		return "content"
	case "Question":
		return "question"
	case "OrderID":
		return "orderId"
	case "Amount":
		return "amount"
	case "Role":
		return "role"
	default:
		if fe.Field() == "" {
			return "body"
		}
		return strings.ToLower(fe.Field()[:1]) + fe.Field()[1:]
	}
}

func fieldMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "필수입니다."
	case "email":
		return "이메일 형식을 확인하세요."
	case "min":
		if fe.Kind() == reflect.String {
			return fe.Param() + "자 이상 입력하세요."
		}
		return fe.Param() + " 이상이어야 합니다."
	case "max":
		if fe.Kind() == reflect.String {
			return fe.Param() + "자 이하로 입력하세요."
		}
		if fe.Kind() == reflect.Slice {
			return fe.Param() + "개 이하로 입력하세요."
		}
		return fe.Param() + " 이하여야 합니다."
	case "oneof":
		return "허용된 값이 아닙니다."
	default:
		return "값을 확인하세요."
	}
}
