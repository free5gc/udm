package sbi

import "strings"

func (s *Server) getUEAuthenticationRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.Processor().HandleIndex,
		},

		{
			"ConfirmAuth",
			strings.ToUpper("Post"),
			"/:supi/auth-events",
			s.Processor().HandleConfirmAuth,
		},
	}
}
