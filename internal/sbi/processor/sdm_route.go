package processor

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Route struct {
	// Name is the name of this Route.
	Name string
	// Method is the string for the HTTP method. ex) GET, POST etc..
	Method string
	// Pattern is the pattern of the URI.
	Pattern string
	// HandlerFunc is the handler function of this route.
	HandlerFunc gin.HandlerFunc
}

func (p *Processor) OneLayerPathHandlerFunc(c *gin.Context) {
	supi := c.Param("supi")
	oneLayerPathRouter := p.getOneLayerRoutes()
	for _, route := range oneLayerPathRouter {
		if strings.Contains(route.Pattern, supi) && route.Method == c.Request.Method {
			route.HandlerFunc(c)
			return
		}
	}

	// special case for :supi
	if c.Request.Method == strings.ToUpper("Get") {
		p.HandleGetSupi(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func (p *Processor) TwoLayerPathHandlerFunc(c *gin.Context) {
	supi := c.Param("supi")
	op := c.Param("subscriptionId")

	// for "/shared-data-subscriptions/:subscriptionId"
	if supi == "shared-data-subscriptions" && strings.ToUpper("Delete") == c.Request.Method {
		p.HandleUnsubscribeForSharedData(c)
		return
	}

	// for "/shared-data-subscriptions/:subscriptionId"
	if supi == "shared-data-subscriptions" && strings.ToUpper("Patch") == c.Request.Method {
		p.HandleModifyForSharedData(c)
		return
	}

	// for "/:gpsi/id-translation-result"
	if op == "id-translation-result" && strings.ToUpper("Get") == c.Request.Method {
		c.Params = append(c.Params, gin.Param{Key: "gpsi", Value: c.Param("supi")})
		p.HandleGetIdTranslationResult(c)
		return
	}

	twoLayerPathRouter := p.getTwoLayerRoutes()
	for _, route := range twoLayerPathRouter {
		if strings.Contains(route.Pattern, op) && route.Method == c.Request.Method {
			route.HandlerFunc(c)
			return
		}
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func (p *Processor) ThreeLayerPathHandlerFunc(c *gin.Context) {
	op := c.Param("subscriptionId")

	// for "/:supi/sdm-subscriptions/:subscriptionId"
	if op == "sdm-subscriptions" && strings.ToUpper("Delete") == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "supi", Value: c.Param("supi")})
		tmpParams = append(tmpParams, gin.Param{Key: "subscriptionId", Value: c.Param("thirdLayer")})
		c.Params = tmpParams
		p.HandleUnsubscribe(c)
		return
	}

	// for "/:supi/am-data/sor-ack"
	if op == "am-data" && strings.ToUpper("Put") == c.Request.Method {
		p.HandleInfo(c)
		return
	}

	// for "/:supi/sdm-subscriptions/:subscriptionId"
	if op == "sdm-subscriptions" && strings.ToUpper("Patch") == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "supi", Value: c.Param("supi")})
		tmpParams = append(tmpParams, gin.Param{Key: "subscriptionId", Value: c.Param("thirdLayer")})
		c.Params = tmpParams
		p.HandleModify(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func (p *Processor) getOneLayerRoutes() []Route {
	return []Route{
		{
			"GetSupi",
			strings.ToUpper("Get"),
			"/:supi",
			p.HandleGetSupi,
		},

		{
			"GetSharedData",
			strings.ToUpper("Get"),
			"/shared-data",
			p.HandleGetSharedData,
		},

		{
			"SubscribeToSharedData",
			strings.ToUpper("Post"),
			"/shared-data-subscriptions",
			p.HandleSubscribeToSharedData,
		},
	}
}

func (p *Processor) getTwoLayerRoutes() []Route {
	return []Route{
		{
			"GetAmData",
			strings.ToUpper("Get"),
			"/:supi/am-data",
			p.HandleGetAmData,
		},

		{
			"GetSmfSelectData",
			strings.ToUpper("Get"),
			"/:supi/smf-select-data",
			p.HandleGetSmfSelectData,
		},

		{
			"GetSmsMngData",
			strings.ToUpper("Get"),
			"/:supi/sms-mng-data",
			p.HandleGetSmsMngData,
		},

		{
			"GetSmsData",
			strings.ToUpper("Get"),
			"/:supi/sms-data",
			p.HandleGetSmsData,
		},

		{
			"GetSmData",
			strings.ToUpper("Get"),
			"/:supi/sm-data",
			p.HandleGetSmData,
		},

		{
			"GetNssai",
			strings.ToUpper("Get"),
			"/:supi/nssai",
			p.HandleGetNssai,
		},

		{
			"Subscribe",
			strings.ToUpper("Post"),
			"/:supi/sdm-subscriptions",
			p.HandleSubscribe,
		},

		{
			"GetTraceData",
			strings.ToUpper("Get"),
			"/:supi/trace-data",
			p.HandleGetTraceData,
		},

		{
			"GetUeContextInSmfData",
			strings.ToUpper("Get"),
			"/:supi/ue-context-in-smf-data",
			p.HandleGetUeContextInSmfData,
		},

		{
			"GetUeContextInSmsfData",
			strings.ToUpper("Get"),
			"/:supi/ue-context-in-smsf-data",
			p.HandleGetUeContextInSmsfData,
		},
	}
}
