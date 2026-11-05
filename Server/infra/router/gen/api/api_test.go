package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/infra/router/gen/schema"
	"github.com/labstack/echo/v5"
)

type fieldServer struct {
	ServerInterface
	fieldID string
}

func (s *fieldServer) GetField(ctx *echo.Context, fieldID FieldId) error {
	s.fieldID = fieldID
	return ctx.NoContent(http.StatusNoContent)
}

func TestEchoRoutesFieldRequest(t *testing.T) {
	server := &fieldServer{}
	router := echo.New()
	RegisterHandlersWithBaseURL(router, server, "/api")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/field/field-1", nil))
	if response.Code != http.StatusNoContent || server.fieldID != "field-1" {
		t.Fatalf("status = %d, field ID = %q", response.Code, server.fieldID)
	}
}

func TestFieldAndStageSchemas(t *testing.T) {
	data := []byte(`{
		"variants": [{
			"width": 1, "height": 1, "terrain": [["land"]],
			"objects": [{"type": "goal", "position": {"x": 0, "y": 0}}],
			"robot": {"position": {"x": 0, "y": 0}, "direction": "up"},
			"drillLimit": 0
		}],
		"clearCondition": {"type": "reach_goal"}
	}`)

	var stage Stage
	if err := json.Unmarshal(data, &stage); err != nil {
		t.Fatal(err)
	}
	if len(stage.Variants) != 1 || stage.Variants[0].Terrain[0][0] != schema.TerrainLand {
		t.Fatalf("unexpected field: %+v", stage.Variants)
	}
	goal := stage.Variants[0].Objects[0].Goal
	if goal == nil || goal.Type != schema.GoalType || goal.Position.X != 0 {
		t.Fatalf("unexpected goal: %+v", goal)
	}
	condition := stage.ClearCondition.ReachGoal
	if condition == nil || condition.Type != schema.ReachGoalType {
		t.Fatalf("unexpected clear condition: %+v", condition)
	}
	encoded, err := json.Marshal(stage)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip Stage
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if roundTrip.Variants[0].Objects[0].Goal == nil || roundTrip.ClearCondition.ReachGoal == nil {
		t.Fatalf("union variants lost on round trip: %+v", roundTrip)
	}
}

func TestUnknownSchemaVariant(t *testing.T) {
	for _, target := range []any{&schema.FieldObject{}, &schema.ClearCondition{}} {
		if err := json.Unmarshal([]byte(`{"type":"unknown"}`), target); err == nil {
			t.Errorf("expected %T to reject an unknown variant", target)
		}
	}
}
