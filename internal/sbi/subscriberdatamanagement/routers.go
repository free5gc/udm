/*
 * Nudm_SDM
 *
 * Nudm Subscriber Data Management Service
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package subscriberdatamanagement

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/udm/pkg/factory"
	logger_util "github.com/free5gc/util/logger"
)

// Route is the information for every URI.
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

// Routes is the list of the generated Route.
type Routes []Route

// NewRouter returns a new router.
func NewRouter() *gin.Engine {
	router := logger_util.NewGinWithLogrus(logger.GinLog)
	AddService(router)
	return router
}

func oneLayerPathHandlerFunc(c *gin.Context) {
	supi := c.Param("supi")
	for _, route := range oneLayerPathRouter {
		if strings.Contains(route.Pattern, supi) && route.Method == c.Request.Method {
			route.HandlerFunc(c)
			return
		}
	}

	// special case for :supi
	if c.Request.Method == strings.ToUpper("Get") {
		HTTPGetSupi(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func twoLayerPathHandlerFunc(c *gin.Context) {
	supi := c.Param("supi")
	op := c.Param("subscriptionId")

	// for "/shared-data-subscriptions/:subscriptionId"
	if supi == "shared-data-subscriptions" && strings.ToUpper("Delete") == c.Request.Method {
		HTTPUnsubscribeForSharedData(c)
		return
	}

	// for "/shared-data-subscriptions/:subscriptionId"
	if supi == "shared-data-subscriptions" && strings.ToUpper("Patch") == c.Request.Method {
		HTTPModifyForSharedData(c)
		return
	}

	// for "/:gpsi/id-translation-result"
	if op == "id-translation-result" && strings.ToUpper("Get") == c.Request.Method {
		c.Params = append(c.Params, gin.Param{Key: "gpsi", Value: c.Param("supi")})
		HTTPGetIdTranslationResult(c)
		return
	}

	for _, route := range twoLayerPathRouter {
		if strings.Contains(route.Pattern, op) && route.Method == c.Request.Method {
			route.HandlerFunc(c)
			return
		}
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func threeLayerPathHandlerFunc(c *gin.Context) {
	op := c.Param("subscriptionId")

	// for "/:supi/sdm-subscriptions/:subscriptionId"
	if op == "sdm-subscriptions" && strings.ToUpper("Delete") == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "supi", Value: c.Param("supi")})
		tmpParams = append(tmpParams, gin.Param{Key: "subscriptionId", Value: c.Param("thirdLayer")})
		c.Params = tmpParams
		HTTPUnsubscribe(c)
		return
	}

	// for "/:supi/am-data/sor-ack"
	if op == "am-data" && strings.ToUpper("Put") == c.Request.Method {
		HTTPInfo(c)
		return
	}

	// for "/:supi/sdm-subscriptions/:subscriptionId"
	if op == "sdm-subscriptions" && strings.ToUpper("Patch") == c.Request.Method {
		var tmpParams gin.Params
		tmpParams = append(tmpParams, gin.Param{Key: "supi", Value: c.Param("supi")})
		tmpParams = append(tmpParams, gin.Param{Key: "subscriptionId", Value: c.Param("thirdLayer")})
		c.Params = tmpParams
		HTTPModify(c)
		return
	}

	c.String(http.StatusNotFound, "404 page not found")
}

func AddService(engine *gin.Engine) *gin.RouterGroup {
	group := engine.Group(factory.UdmSdmResUriPrefix)

	for _, route := range routes {
		switch route.Method {
		case "GET":
			group.GET(route.Pattern, route.HandlerFunc)
		case "POST":
			group.POST(route.Pattern, route.HandlerFunc)
		case "PUT":
			group.PUT(route.Pattern, route.HandlerFunc)
		case "DELETE":
			group.DELETE(route.Pattern, route.HandlerFunc)
		case "PATCH":
			group.PATCH(route.Pattern, route.HandlerFunc)
		}
	}

	oneLayerPath := "/:supi"
	group.Any(oneLayerPath, oneLayerPathHandlerFunc)

	twoLayerPath := "/:supi/:subscriptionId"
	group.Any(twoLayerPath, twoLayerPathHandlerFunc)

	threeLayerPath := "/:supi/:subscriptionId/:thirdLayer"
	group.Any(threeLayerPath, threeLayerPathHandlerFunc)

	return group
}

// Index is the index handler.
func Index(c *gin.Context) {
	c.String(http.StatusOK, "Hello World!")
}

func authorizationCheck(c *gin.Context) error {
	token := c.Request.Header.Get("Authorization")
	return udm_context.GetSelf().AuthorizationCheck(token, "nudm-sdm")
}

var routes = Routes{
	{
		"Index",
		"GET",
		"/",
		Index,
	},
}

/*
var specialRouter = Routes{
	{
		"GetIdTranslationResult",
		strings.ToUpper("Get"),
		"/:gpsi/id-translation-result",
		HTTPGetIdTranslationResult,
	},

	{
		"UnsubscribeForSharedData",
		strings.ToUpper("Delete"),
		"/shared-data-subscriptions/:subscriptionId",
		HTTPUnsubscribeForSharedData,
	},

	{
		"ModifyForSharedData",
		strings.ToUpper("Patch"),
		"/shared-data-subscriptions/:subscriptionId",
		HTTPModifyForSharedData,
	},
}
*/

var oneLayerPathRouter = Routes{
	{
		"GetSupi",
		strings.ToUpper("Get"),
		"/:supi",
		HTTPGetSupi,
	},

	{
		"GetSharedData",
		strings.ToUpper("Get"),
		"/shared-data",
		HTTPGetSharedData,
	},

	{
		"SubscribeToSharedData",
		strings.ToUpper("Post"),
		"/shared-data-subscriptions",
		HTTPSubscribeToSharedData,
	},
}

var twoLayerPathRouter = Routes{
	{
		"GetAmData",
		strings.ToUpper("Get"),
		"/:supi/am-data",
		HTTPGetAmData,
	},

	{
		"GetSmfSelectData",
		strings.ToUpper("Get"),
		"/:supi/smf-select-data",
		HTTPGetSmfSelectData,
	},

	{
		"GetSmsMngData",
		strings.ToUpper("Get"),
		"/:supi/sms-mng-data",
		HTTPGetSmsMngData,
	},

	{
		"GetSmsData",
		strings.ToUpper("Get"),
		"/:supi/sms-data",
		HTTPGetSmsData,
	},

	{
		"GetSmData",
		strings.ToUpper("Get"),
		"/:supi/sm-data",
		HTTPGetSmData,
	},

	{
		"GetNssai",
		strings.ToUpper("Get"),
		"/:supi/nssai",
		HTTPGetNssai,
	},

	{
		"Subscribe",
		strings.ToUpper("Post"),
		"/:supi/sdm-subscriptions",
		HTTPSubscribe,
	},

	{
		"GetTraceData",
		strings.ToUpper("Get"),
		"/:supi/trace-data",
		HTTPGetTraceData,
	},

	{
		"GetUeContextInSmfData",
		strings.ToUpper("Get"),
		"/:supi/ue-context-in-smf-data",
		HTTPGetUeContextInSmfData,
	},

	{
		"GetUeContextInSmsfData",
		strings.ToUpper("Get"),
		"/:supi/ue-context-in-smsf-data",
		HTTPGetUeContextInSmsfData,
	},
}

/*
var threeLayerPathRouter = Routes{
	{
		"Unsubscribe",
		strings.ToUpper("Delete"),
		"/:supi/sdm-subscriptions/:subscriptionId",
		HTTPUnsubscribe,
	},

	{
		"Info",
		strings.ToUpper("Put"),
		"/:supi/am-data/sor-ack",
		HTTPInfo,
	},

	{
		"PutUpuAck",
		strings.ToUpper("Put"),
		"/:supi/am-data/upu-ack",
		HTTPPutUpuAck,
	},

	{
		"Modify",
		strings.ToUpper("Patch"),
		"/:supi/sdm-subscriptions/:subscriptionId",
		HTTPModify,
	},
}

var routesBackup = Routes{
	{
		"Index",
		"GET",
		"/",
		Index,
	},

	{
		"GetAmData",
		strings.ToUpper("Get"),
		"/:supi/am-data",
		HTTPGetAmData,
	},

	{
		"Info",
		strings.ToUpper("Put"),
		"/:supi/am-data/sor-ack",
		HTTPInfo,
	},

	{
		"GetSupi",
		strings.ToUpper("Get"),
		"/:supi",
		HTTPGetSupi,
	},

	{
		"GetSharedData",
		strings.ToUpper("Get"),
		"/shared-data",
		HTTPGetSharedData,
	},

	{
		"GetSmfSelectData",
		strings.ToUpper("Get"),
		"/:supi/smf-select-data",
		HTTPGetSmfSelectData,
	},

	{
		"GetSmsMngData",
		strings.ToUpper("Get"),
		"/:supi/sms-mng-data",
		HTTPGetSmsMngData,
	},

	{
		"GetSmsData",
		strings.ToUpper("Get"),
		"/:supi/sms-data",
		HTTPGetSmsData,
	},

	{
		"GetSmData",
		strings.ToUpper("Get"),
		"/:supi/sm-data",
		HTTPGetSmData,
	},

	{
		"GetNssai",
		strings.ToUpper("Get"),
		"/:supi/nssai",
		HTTPGetNssai,
	},

	{
		"Subscribe",
		strings.ToUpper("Post"),
		"/:supi/sdm-subscriptions",
		HTTPSubscribe,
	},

	{
		"SubscribeToSharedData",
		strings.ToUpper("Post"),
		"/shared-data-subscriptions",
		HTTPSubscribeToSharedData,
	},

	{
		"Unsubscribe",
		strings.ToUpper("Delete"),
		"/:supi/sdm-subscriptions/:subscriptionId",
		HTTPUnsubscribe,
	},

	{
		"UnsubscribeForSharedData",
		strings.ToUpper("Delete"),
		"/shared-data-subscriptions/:subscriptionId",
		HTTPUnsubscribeForSharedData,
	},

	{
		"GetTraceData",
		strings.ToUpper("Get"),
		"/:supi/trace-data",
		HTTPGetTraceData,
	},

	{
		"GetUeContextInSmfData",
		strings.ToUpper("Get"),
		"/:supi/ue-context-in-smf-data",
		HTTPGetUeContextInSmfData,
	},

	{
		"GetUeContextInSmsfData",
		strings.ToUpper("Get"),
		"/:supi/ue-context-in-smsf-data",
		HTTPGetUeContextInSmsfData,
	},

	{
		"GetIdTranslationResult",
		strings.ToUpper("Get"),
		"/:gpsi/id-translation-result",
		HTTPGetIdTranslationResult,
	},
}
*/
