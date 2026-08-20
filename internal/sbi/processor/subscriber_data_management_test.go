package processor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSdmSystemFailureProblemDetailsDoesNotExposeInternalError(t *testing.T) {
	err := errors.New(`parse "http://udr.internal:8000/nudr-dr/v2/` +
		`subscription-data/imsi-001": invalid control character in URL`)

	problemDetails := newSdmSystemFailureProblemDetails(err)

	require.Equal(t, "Upstream request failed", problemDetails.Detail)
	require.NotContains(t, problemDetails.Detail, "udr.internal")
	require.NotContains(t, problemDetails.Detail, "imsi-001")
}
