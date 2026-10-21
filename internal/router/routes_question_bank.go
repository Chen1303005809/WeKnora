package router

import (
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterQuestionBankRoutes exposes published banks and member-owned attempts
// to Viewer+, while keeping every bank mutation and answer-key read Admin-only.
func RegisterQuestionBankRoutes(r *gin.RouterGroup, h *handler.QuestionBankHandler, g *rbacGuards) {
	if h == nil {
		return
	}
	group := g.apiKeyGroup(r.Group("/question-banks"), apiKeyFullAccess())
	group.GET("", g.Viewer(), h.ListPublished)
	group.GET("/manage", g.Admin(), h.ListManage)
	group.POST("", g.Admin(), h.Create)
	group.GET("/:id", g.Admin(), h.GetDraft)
	group.PATCH("/:id", g.Admin(), h.UpdateBank)
	group.PATCH("/:id/questions/:questionID", g.Admin(), h.UpdateQuestion)
	group.POST("/:id/publish", g.Admin(), h.Publish)
	group.POST("/:id/attempts", g.Viewer(), h.StartAttempt)
	group.GET("/:id/attempts", g.Viewer(), h.ListMyAttempts)
	group.POST("/attempts/:attemptID/submit", g.Viewer(), h.SubmitAttempt)
	group.GET("/attempts/:attemptID", g.Viewer(), h.GetMyAttempt)
}
