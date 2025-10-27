package utils
import "net/http"
// Response 统一响应格式
func Response(w http.ResponseWriter, code int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"code": code,
		"msg":  msg,
	}
	if data != nil {
		response["data"] = data
	}
}
// ErrorResponse 错误响应
func ErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
}
