package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type recordingEngine struct {
	calls int
}

func (e *recordingEngine) Execute(_ context.Context, _ studioapp.AgentRequest, _ studioapp.AgentSink) error {
	e.calls++
	return nil
}

func TestDispatchEngineRejectsRunWithoutConfiguredModel(t *testing.T) {
	t.Parallel()
	online := &recordingEngine{}

	err := (studioapp.DispatchEngine{Online: online}).Execute(context.Background(), studioapp.AgentRequest{
		Run: &domain.Run{},
	}, nil)

	require.ErrorContains(t, err, "model is required")
	require.Zero(t, online.calls)
}

func TestDispatchEngineUsesOnlineWithConfiguredModel(t *testing.T) {
	t.Parallel()
	online := &recordingEngine{}

	err := (studioapp.DispatchEngine{Online: online}).Execute(context.Background(), studioapp.AgentRequest{
		Run: &domain.Run{ModelConfigID: "model_01"},
	}, nil)

	require.NoError(t, err)
	require.Equal(t, 1, online.calls)
}
