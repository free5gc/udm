package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PutUpuAck - Nudm_Sdm Info for UPU service operation
func (p *Processor) HandlePutUpuAck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
