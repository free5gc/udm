package consumer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/pkg/app"
	"github.com/free5gc/udm/pkg/factory"
)

func newTestConsumer(t *testing.T, ctx *udm_context.UDMContext) *Consumer {
	t.Helper()

	controller := gomock.NewController(t)
	mockApp := app.NewMockApp(controller)
	mockApp.EXPECT().Context().Return(ctx).AnyTimes()
	mockApp.EXPECT().Config().Return(&factory.Config{
		Configuration: &factory.Configuration{},
	}).AnyTimes()

	testConsumer, err := NewConsumer(mockApp)
	require.NoError(t, err)

	return testConsumer
}
