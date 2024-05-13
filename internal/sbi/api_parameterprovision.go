package sbi

import "strings"

func (s *Server) getParameterProvisionRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.Processor().HandleIndex,
		},

		{
			"Update",
			strings.ToUpper("Patch"),
			"/:gpsi/pp-data",
			s.Processor().HandleUpdate,
		},
	}
}
