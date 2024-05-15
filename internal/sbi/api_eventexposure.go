package sbi

import (
	"strings"
)

func (s *Server) getEventExposureRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.Processor().HandleIndex,
		},

		{
			"HTTPCreateEeSubscription",
			strings.ToUpper("Post"),
			"/:ueIdentity/ee-subscriptions",
			s.Processor().HandleCreateEeSubscription,
		},

		{
			"HTTPDeleteEeSubscription",
			strings.ToUpper("Delete"),
			"/:ueIdentity/ee-subscriptions/:subscriptionId",
			s.Processor().HandleDeleteEeSubscription,
		},

		{
			"HTTPUpdateEeSubscription",
			strings.ToUpper("Patch"),
			"/:ueIdentity/ee-subscriptions/:subscriptionId",
			s.Processor().HandleUpdateEeSubscription,
		},
	}
}
