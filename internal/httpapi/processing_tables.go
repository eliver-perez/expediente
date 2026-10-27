package httpapi

import (
	"gestor-documental/internal/domain"
	"gestor-documental/internal/libraries"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) processingTable(w http.ResponseWriter, r *http.Request, p domain.Principal) {
	q := r.URL.Query()
	page, size := 1, 10
	var err error
	if q.Has("page") {
		page, err = strconv.Atoi(q.Get("page"))
		if err != nil {
			page = 0
		}
	}
	if q.Has("page_size") {
		size, err = strconv.Atoi(q.Get("page_size"))
		if err != nil {
			size = 0
		}
	}
	sort, direction := q.Get("sort"), q.Get("direction")
	if sort == "" {
		sort = "date"
	}
	if direction == "" {
		direction = "desc"
	}
	f := libraries.TableFilter{Query: q.Get("q"), State: q.Get("state"), Kind: q.Get("kind"), From: q.Get("from"), Until: q.Get("until"), Sort: sort, Direction: direction, Page: page, Size: size}
	if strings.HasSuffix(r.URL.Path, "/file-skips") {
		result, err := s.libraries.FileSkips(r.Context(), p, r.PathValue("library"), f)
		s.libraryResult(w, r, 200, result, err)
	} else {
		result, err := s.libraries.JobHistory(r.Context(), p, r.PathValue("library"), f)
		s.libraryResult(w, r, 200, result, err)
	}
}
