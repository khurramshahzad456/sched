package server

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func Run(router *gin.Engine) {
	port := os.Getenv("PORT")
	addr := ":8080"
	if port != "" {
		addr = ":" + port
	}
	if err := router.Run(addr); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
