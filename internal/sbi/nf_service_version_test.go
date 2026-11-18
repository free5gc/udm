package sbi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/free5gc/openapi/models"
	udm_context "github.com/free5gc/udm/internal/context"
)

// Each service that the NF profile advertises must be served at the version
// it advertises: an NF that discovers the UDM through the NRF builds its URIs
// from apiVersionInUri (TS 29.510 §6.1.6.2.19).
func TestNFServiceVersionsMatchTheRouter(t *testing.T) {
	routes := newRouter(&Server{}).Routes()
	udmCtx := &udm_context.UDMContext{
		NfService: make(map[models.Nrf_NFMgmt_ServiceName]models.Nrf_NFMgmt_NFService),
	}
	// The services that the configuration accepts in serviceNameList.
	services := []string{"nudm-sdm", "nudm-uecm", "nudm-ueau", "nudm-ee", "nudm-pp"}
	udmCtx.InitNFService(services)

	for _, name := range services {
		t.Run(name, func(t *testing.T) {
			service, ok := udmCtx.NfService[models.Nrf_NFMgmt_ServiceName(name)]
			require.True(t, ok)
			require.Len(t, service.Versions, 1)
			version := service.Versions[0]

			prefix := "/" + name + "/" + version.ApiVersionInUri + "/"
			served := false
			for _, route := range routes {
				if strings.HasPrefix(route.Path, prefix) {
					served = true
					break
				}
			}
			require.True(t, served, "advertised as %s, but nothing is served under %s",
				version.ApiVersionInUri, prefix)

			// The version in the URI is "v" and the MAJOR of the full version
			// (TS 29.501 §4.3.1.3).
			major := strings.TrimPrefix(version.ApiVersionInUri, "v")
			require.True(t, strings.HasPrefix(version.ApiFullVersion, major+"."),
				"apiFullVersion %s does not have the MAJOR of %s", version.ApiFullVersion, version.ApiVersionInUri)
		})
	}
}
