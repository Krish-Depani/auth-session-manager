package main

import (
	"log"
	"net/http"

	"github.com/Krish-Depani/auth-session-manager/config"
	"github.com/Krish-Depani/auth-session-manager/controllers"
	"github.com/Krish-Depani/auth-session-manager/database"
	"github.com/Krish-Depani/auth-session-manager/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	env, err := config.LoadEnv()
	if err != nil {
		log.Fatal("Error loading .env:", err)
	}

	pgClient, err := database.NewPostgresClient(env.DBHost, env.DBUser, env.DBPassword, env.DBName, env.DBPort)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	redisClient, err := database.GetRedisClient(env.RedisAddr, env.RedisPass, env.RedisDB)
	if err != nil {
		log.Fatal("Error connecting to redis:", err)
	}

	authController := controllers.NewAuthController(pgClient, redisClient)

	r := gin.Default()
	routes.SetupRoutes(r, authController)

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "Hello, World!"})
	})

	if err := r.Run(":3000"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
