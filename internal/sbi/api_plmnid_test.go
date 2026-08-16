package sbi

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// GHSA-42jf-j68x-57gx: getPlmnIDStruct parses plmn-id as JSON with no
// length limit on mcc/mnc. An oversized mcc propagates unsanitized into
// the path of the internal outbound UDR request, causing a ~10s timeout
// and an error response disclosing the UDR's internal hostname/port/API
// path — the same disclosure class as CVE-2026-42459 via payload size.
func TestGetPlmnIDStructRejectsUnboundedMccMnc(t *testing.T) {
	s := &Server{}

	cases := []struct {
		name    string
		plmnID  string
		wantBad bool
	}{
		{
			name:    "oversized mcc",
			plmnID:  `{"mcc":"` + strings.Repeat("A", 5000) + `","mnc":"77"}`,
			wantBad: true,
		},
		{
			name:    "oversized mnc",
			plmnID:  `{"mcc":"222","mnc":"` + strings.Repeat("7", 5000) + `"}`,
			wantBad: true,
		},
		{
			name:    "non-digit mcc",
			plmnID:  `{"mcc":"22a","mnc":"77"}`,
			wantBad: true,
		},
		{
			name:    "mcc too short",
			plmnID:  `{"mcc":"22","mnc":"77"}`,
			wantBad: true,
		},
		{
			name:    "mnc too short",
			plmnID:  `{"mcc":"222","mnc":"7"}`,
			wantBad: true,
		},
		{
			name:   "valid mcc/mnc",
			plmnID: `{"mcc":"222","mnc":"77"}`,
		},
		{
			name:   "valid 3-digit mnc",
			plmnID: `{"mcc":"222","mnc":"077"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := url.Values{"plmn-id": []string{tc.plmnID}}
			structure, problem := s.getPlmnIDStruct(values)

			if tc.wantBad {
				if problem == nil || problem.Status != http.StatusBadRequest {
					t.Fatalf("expected 400 problem, got structure=%+v problem=%+v",
						structure, problem)
				}
				return
			}
			if problem != nil || structure == nil {
				t.Fatalf("expected valid parse, got problem=%+v", problem)
			}
		})
	}
}
