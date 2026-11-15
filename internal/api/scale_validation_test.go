package api

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScaleRejectsMissingOrInvalidReplicas(t *testing.T) {
	for _, value := range []string{"", `,"replicas":null`, `,"replicas":-1`, `,"replicas":1.5`} {
		t.Run(value, func(t *testing.T) {
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = httptest.NewRequest("POST", "/workloads/scale", strings.NewReader(`{"kind":"deployment","namespace":"default","name":"app"`+value+`}`))
			ctx.Request.Header.Set("Content-Type", "application/json")
			// No Kubernetes client: invalid input must return before any cluster access.
			(&Handler{}).scaleWorkload(ctx)
			if response.Code != 400 {
				t.Fatalf("status = %d: %s", response.Code, response.Body.String())
			}
		})
	}
}
