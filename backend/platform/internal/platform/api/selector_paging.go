package api

import (
	"net/http"
	"strconv"
)

func selectorPaging(r *http.Request, body map[string]any) error {
	if query := r.URL.Query().Get("query"); len([]rune(query)) > 256 {
		return fail(400, "INVALID_REQUEST", "搜索内容不能超过256个字符")
	}
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		if len([]rune(cursor)) > 2048 {
			return fail(400, "INVALID_REQUEST", "分页游标过长")
		}
		body["cursor"] = cursor
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return fail(400, "INVALID_REQUEST", "每页数量应为1到100")
		}
		body["limit"] = limit
	}
	return nil
}
