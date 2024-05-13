package sbi

import "strings"

func (s *Server) getUEContextManagementRoutes() []Route {
	return []Route{
		{
			"Index",
			"GET",
			"/",
			s.Processor().HandleIndex,
		},

		{
			"GetAmf3gppAccess",
			strings.ToUpper("Get"),
			"/:ueId/registrations/amf-3gpp-access",
			s.Processor().HandleGetAmf3gppAccess,
		},

		{
			"GetAmfNon3gppAccess",
			strings.ToUpper("Get"),
			"/:ueId/registrations/amf-non-3gpp-access",
			s.Processor().HandleGetAmfNon3gppAccess,
		},

		{
			"RegistrationAmf3gppAccess",
			strings.ToUpper("Put"),
			"/:ueId/registrations/amf-3gpp-access",
			s.Processor().HandleRegistrationAmf3gppAccess,
		},

		{
			"Register",
			strings.ToUpper("Put"),
			"/:ueId/registrations/amf-non-3gpp-access",
			s.Processor().HandleRegistrationAmfNon3gppAccess,
		},

		{
			"UpdateAmf3gppAccess",
			strings.ToUpper("Patch"),
			"/:ueId/registrations/amf-3gpp-access",
			s.Processor().HandleUpdateAmf3gppAccess,
		},

		{
			"UpdateAmfNon3gppAccess",
			strings.ToUpper("Patch"),
			"/:ueId/registrations/amf-non-3gpp-access",
			s.Processor().HandleUpdateAmfNon3gppAccess,
		},

		{
			"DeregistrationSmfRegistrations",
			strings.ToUpper("Delete"),
			"/:ueId/registrations/smf-registrations/:pduSessionId",
			s.Processor().HandleDeregistrationSmfRegistrations,
		},

		{
			"RegistrationSmfRegistrations",
			strings.ToUpper("Put"),
			"/:ueId/registrations/smf-registrations/:pduSessionId",
			s.Processor().HandleRegistrationSmfRegistrations,
		},

		{
			"GetSmsf3gppAccess",
			strings.ToUpper("Get"),
			"/:ueId/registrations/smsf-3gpp-access",
			s.Processor().HandleGetSmsf3gppAccess,
		},

		{
			"DeregistrationSmsf3gppAccess",
			strings.ToUpper("Delete"),
			"/:ueId/registrations/smsf-3gpp-access",
			s.Processor().HandleDeregistrationSmsf3gppAccess,
		},

		{
			"DeregistrationSmsfNon3gppAccess",
			strings.ToUpper("Delete"),
			"/:ueId/registrations/smsf-non-3gpp-access",
			s.Processor().HandleDeregistrationSmsfNon3gppAccess,
		},

		{
			"GetSmsfNon3gppAccess",
			strings.ToUpper("Get"),
			"/:ueId/registrations/smsf-non-3gpp-access",
			s.Processor().HandleGetSmsfNon3gppAccess,
		},

		{
			"UpdateSMSFReg3GPP",
			strings.ToUpper("Put"),
			"/:ueId/registrations/smsf-3gpp-access",
			s.Processor().HandleUpdateSMSFReg3GPP,
		},

		{
			"RegistrationSmsfNon3gppAccess",
			strings.ToUpper("Put"),
			"/:ueId/registrations/smsf-non-3gpp-access",
			s.Processor().HandleRegistrationSmsfNon3gppAccess,
		},
	}
}
