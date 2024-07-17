package processor

import (
	"net/http"

	Nudr_DataRepository "github.com/free5gc/openapi/udr/DataRepository"
	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
)

func (p *Processor) UpdateProcedure(c *gin.Context,
	updateRequest models.PpData,
	gpsi string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.ServiceName_NUDR_DR, models.NrfNfManagementNfType_UDR)
	if err != nil {
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(gpsi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	var modifyPpDataRequest Nudr_DataRepository.ModifyPpDataRequest
	modifyPpDataRequest.UeId = &gpsi
	modifyPpDataRsp, err := clientAPI.ProvisionedParameterDataDocumentApi.ModifyPpData(ctx, &modifyPpDataRequest)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	if modifyPpDataRsp.PatchResult.Report != nil {
		c.JSON(http.StatusOK, modifyPpDataRsp)
		return
	}

	c.Status(http.StatusNoContent)
}
