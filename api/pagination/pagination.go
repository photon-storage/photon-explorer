package pagination

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	defaultStartStr = "0"
	defaultLimitStr = "10"
	maxPageLimit    = 100
)

type Query struct {
	Start int
	Limit int
}

type Result struct {
	Data  any   `json:"data"`
	Total int64 `json:"total"`
}

// Response is the response for pagination query request
type Response struct {
	Code int `json:"code"`
	*Result
	Links links `json:"_links"`
}

type links struct {
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

func Parse(c *gin.Context) (*Query, error) {
	startStr := c.DefaultQuery("start", defaultStartStr)
	limitStr := c.DefaultQuery("limit", defaultLimitStr)

	start, err := strconv.ParseUint(startStr, 10, 64)
	if err != nil {
		return nil, err
	}

	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil {
		return nil, err
	}

	if limit > maxPageLimit {
		limit = maxPageLimit
	}

	return &Query{
		Start: int(start),
		Limit: int(limit),
	}, nil
}

// GetLinks returns the prev and next links of the request
func GetLinks(ctx *gin.Context, total int64, q *Query) links {
	url := fmt.Sprintf("%v", ctx.Request.URL)
	baseURL := strings.Split(url, "?")[0]
	l := links{}
	if int64(q.Start+q.Limit) <= total {
		l.Next = fmt.Sprintf(
			"%s?limit=%d&start=%d",
			baseURL,
			q.Limit,
			q.Start+q.Limit,
		)
	}

	if q.Start != 0 {
		prevStart := q.Start - q.Limit
		if prevStart < 0 || q.Start < q.Limit {
			prevStart = 0
		}

		l.Prev = fmt.Sprintf(
			"%s?limit=%d&start=%d",
			baseURL,
			q.Limit,
			prevStart,
		)
	}

	return l
}
