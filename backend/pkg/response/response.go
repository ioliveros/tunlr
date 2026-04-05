package response

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, envelope{Success: true, Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, envelope{Success: true, Data: data})
}

func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, envelope{Success: false, Error: message})
}
