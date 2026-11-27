package httpapi

import (
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"
	"taotale/backend/internal/persistence/queries"
)

var literalLike = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)

func itemFilters(query url.Values) (queries.PageItemsParams, bool) {
	for _, key := range []string{"q", "status", "difficulty", "favoritesOnly", "topics", "sort"} {
		if len(query[key]) > 1 {
			return queries.PageItemsParams{}, false
		}
	}
	search := strings.TrimSpace(query.Get("q"))
	status, difficulty, sort := query.Get("status"), query.Get("difficulty"), query.Get("sort")
	favorites := query.Get("favoritesOnly")
	if !utf8.ValidString(search) || strings.ContainsRune(search, 0) || utf8.RuneCountInString(query.Get("q")) > 120 ||
		!slices.Contains([]string{"", "all", "draft", "published"}, status) ||
		!slices.Contains([]string{"", "unspecified", "beginner", "intermediate", "advanced"}, difficulty) ||
		!slices.Contains([]string{"", "newest", "oldest", "title_asc", "title_desc"}, sort) ||
		!slices.Contains([]string{"", "true", "false"}, favorites) {
		return queries.PageItemsParams{}, false
	}
	if status == "" {
		status = "all"
	}
	if sort == "" {
		sort = "newest"
	}
	var ids []pgtype.UUID
	if query.Has("topics") {
		seen := make(map[pgtype.UUID]bool)
		for _, raw := range strings.Split(query.Get("topics"), ",") {
			id, ok := parseID(raw)
			if !ok || seen[id] {
				return queries.PageItemsParams{}, false
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if search != "" {
		search = "%" + literalLike.Replace(search) + "%"
	}
	return queries.PageItemsParams{
		TitlePattern: search, Status: status, Difficulty: difficulty, FavoritesOnly: favorites == "true",
		TopicIds: ids, SortOrder: sort,
	}, true
}
