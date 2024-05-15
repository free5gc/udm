package processor

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (p *Processor) GenAuthDataHandlerFunc(c *gin.Context) {
	c.Params = append(c.Params, gin.Param{Key: "supiOrSuci", Value: c.Param("supi")})
	if strings.ToUpper("Post") == c.Request.Method {
		p.HandleGenerateAuthData(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}
