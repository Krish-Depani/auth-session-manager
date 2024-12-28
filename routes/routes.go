package routes

import (
	"github.com/Krish-Depani/auth-session-manager/controllers"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, authController *controllers.AuthController, userController *controllers.UserController) {
	auth := router.Group("/auth")
	{
		auth.POST("/register", authController.Register)
		auth.POST("/login", authController.Login)
		auth.POST("/logout", authController.AuthMiddleware(), authController.Logout)
	}

	user := router.Group("/auth/user")
	{
		user.GET("/me", authController.AuthMiddleware(), userController.GetCurrentUser)
		user.GET("/sessions", authController.AuthMiddleware(), userController.GetActiveSessions)
	}
}
