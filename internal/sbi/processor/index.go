package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (p *Processor) HandleIndex(c *gin.Context) {
	c.String(http.StatusOK, "Hello World!")
}
