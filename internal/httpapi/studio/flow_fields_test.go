package studio_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Mr9esx/Pixoma/internal/httpapi/apitest"
	studioapi "github.com/Mr9esx/Pixoma/internal/httpapi/studio"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	taskpersist "github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/persistence"
)

func TestSessionWorkflowFieldsHTTP(t *testing.T) {
	ctx := context.Background()
	gdb, err := db.Open(db.Options{DSN: "file:" + uuid.NewString() + "?mode=memory&cache=shared"})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(gdb, persistence.Models()...))
	require.NoError(t, db.AutoMigrate(gdb, &taskpersist.TaskRow{}))
	repo := persistence.NewGormRepository(gdb)
	tasks := taskpersist.NewTaskRepository(gdb)
	now := time.Now().UTC()
	session, err := domain.NewSession("session", "account-a", now)
	require.NoError(t, err)
	require.NoError(t, repo.CreateSession(ctx, session))
	node, err := domain.NewFlowNode("operation", session.ID, session.AccountID, domain.FlowNodeOperation, "生成分镜", 0, now)
	require.NoError(t, err)
	node.Outputs = []domain.FlowOutput{{Key: "image", Type: "image", Name: "分镜图", AssetID: "output", AssetVersionID: "version"}}
	require.NoError(t, repo.SaveFlowNode(ctx, node))
	execution, err := domain.NewWorkflowExecution("execution", session.AccountID, session.ID, "run", "tool", "task", "1", node.ID, now)
	require.NoError(t, err)
	execution.InputFields = []domain.FlowPort{{Key: "prompt", Type: "string"}}
	execution.OutputFields = []domain.FlowPort{{Key: "image", Type: "image"}}
	execution.Inputs = []domain.FlowInput{{Key: "prompt", Value: "森林小屋"}}
	require.NoError(t, repo.CreateWorkflowExecution(ctx, execution))
	require.NoError(t, tasks.Create(ctx, runtimedomain.NewPending("task", sharedkernel.SessionID(session.ID), 1, "inputs/task", now)))
	router := chi.NewRouter()
	(&studioapi.Handler{Repo: repo, Tasks: tasks}).Mount(router)
	list := request(t, router, http.MethodGet, "/sessions", nil, session.AccountID)
	require.Equal(t, http.StatusOK, list.Code, list.Body.String())
	var sessions []struct {
		ActiveWorkflowCount int `json:"active_workflow_count"`
	}
	require.NoError(t, json.Unmarshal(apitest.DataBytes(list), &sessions))
	require.Len(t, sessions, 1)
	require.Equal(t, 1, sessions[0].ActiveWorkflowCount)
	detail := request(t, router, http.MethodGet, "/sessions/session", nil, session.AccountID)
	require.Equal(t, http.StatusOK, detail.Code, detail.Body.String())
	var value struct {
		Session struct {
			ActiveWorkflowCount int `json:"active_workflow_count"`
		} `json:"session"`
		Executions []struct {
			InputFields  []domain.FlowPort  `json:"input_fields"`
			OutputFields []domain.FlowPort  `json:"output_fields"`
			Inputs       []domain.FlowInput `json:"inputs"`
			TaskStatus   string             `json:"task_status"`
		} `json:"workflow_executions"`
		Flow struct {
			Nodes []struct {
				Outputs []domain.FlowOutput `json:"outputs"`
			} `json:"nodes"`
		} `json:"flow"`
	}
	require.NoError(t, json.Unmarshal(apitest.DataBytes(detail), &value))
	require.Len(t, value.Executions, 1)
	require.Equal(t, execution.InputFields, value.Executions[0].InputFields)
	require.Equal(t, execution.OutputFields, value.Executions[0].OutputFields)
	require.Equal(t, execution.Inputs, value.Executions[0].Inputs)
	require.Equal(t, "pending", value.Executions[0].TaskStatus)
	require.Len(t, value.Flow.Nodes, 1)
	require.Equal(t, node.Outputs, value.Flow.Nodes[0].Outputs)
	require.Equal(t, 1, value.Session.ActiveWorkflowCount)
	foreign := request(t, router, http.MethodGet, "/sessions/session", nil, "account-b")
	require.Equal(t, http.StatusNotFound, foreign.Code)
}
