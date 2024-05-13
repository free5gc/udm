package sbi

func (s *Server) getSubscriberDataManagementRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.Processor().HandleIndex,
		},
	}
}
