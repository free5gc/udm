package processor

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	Nudr_DataRepository "github.com/free5gc/openapi/udr/DR"
	"github.com/free5gc/util/metrics/sbi"
)

func (p *Processor) UpdateProcedure(c *gin.Context,
	updateRequest models.Udm_PP_PpData,
	gpsi string,
) {
	ctx, pd, err := p.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	clientAPI, err := p.Consumer().CreateUDMClientToUDR(gpsi)
	if err != nil {
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	var modifyPpDataRequest Nudr_DataRepository.ModifyPpDataRequest
	modifyPpDataRequest.UeId = &gpsi
	modifyPpDataRsp, err := clientAPI.ProvisionedParameterDataDocumentApi.ModifyPpData(ctx, &modifyPpDataRequest)
	if err != nil {
		if apiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			if modification_err, ok2 := apiErr.Model().(Nudr_DataRepository.ModifyPpDataError); ok2 {
				problem := modification_err.ProblemDetails
				c.Set(sbi.IN_PB_DETAILS_CTX_STR, problem.Cause)
				c.JSON(int(problem.Status), problem)
				return
			}
		}
		problemDetails := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	if modifyPpDataRsp.PatchResult.Report != nil {
		c.JSON(http.StatusOK, modifyPpDataRsp.PatchResult)
		return
	}

	c.Status(http.StatusNoContent)
}
