package handlers

import (
	"infra-search/search"
	"net/http"

	"github.com/gin-gonic/gin"
)

type searchRequest struct {
	Query string `json:"query" binding:"required"`
}

func SearchHandler(c *gin.Context) {
	var req searchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query は必須です"})
		return
	}

	query := search.BuildQuery(req.Query)
	results, err := search.Search(query)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"results":  []interface{}{},
			"commands": nil,
			"message":  err.Error(),
		})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"results":  []interface{}{},
			"commands": nil,
			"message":  "結果が見つかりませんでした",
		})
		return
	}

	commands, _ := search.Summarize(req.Query, results)
	c.JSON(http.StatusOK, gin.H{
		"results":  results,
		"commands": commands,
	})
}
