package http

import (
	"github.com/gin-gonic/gin"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
)

func RegisterRoutes(r *gin.Engine, sessionUC inbound.SessionUseCase, eventUC inbound.EventStreamUseCase, uiPath string) {
	auth := NewAuthHandler(sessionUC, uiPath)
	sse := NewSSEHandler(eventUC)

	v1 := r.Group("/api/v1")
	{
		// Public auth endpoints
		v1.GET("/auth/login", auth.Login)
		v1.GET("/auth/callback", auth.Callback)
		v1.POST("/auth/backchannel-logout", auth.BackchannelLogout)

		// Protected group
		protected := v1.Group("")
		protected.Use(SessionMiddleware(sessionUC, uiPath))
		{
			protected.POST("/auth/logout", auth.Logout)
			protected.GET("/events/stream", sse.StreamEvents)
		}

		// Placeholder groups for future specs
		configurator := v1.Group("/configurator")
		configurator.Use(SessionMiddleware(sessionUC, uiPath))
		{
			// Future configurator routes
		}

		auditor := v1.Group("/auditor")
		auditor.Use(SessionMiddleware(sessionUC, uiPath))
		{
			// Future auditor routes
		}
	}
}
