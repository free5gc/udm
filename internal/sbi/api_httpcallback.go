package sbi

import "strings"

func (s *Server) getHttpCallBackRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.Processor().HandleIndex,
		},

		{
			"DataChangeNotificationToNF",
			strings.ToUpper("Post"),
			"/sdm-subscriptions",
			s.Processor().HandleDataChangeNotificationToNF,
		},
	}
}
