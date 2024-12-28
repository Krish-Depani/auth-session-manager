package controllers

import (
	"net/http"
	"time"

	"github.com/Krish-Depani/auth-session-manager/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct {
	db *gorm.DB
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{
		db: db,
	}
}

type SessionResponse struct {
	ID             uint      `json:"id"`
	DeviceInfo     string    `json:"device_info"`
	IPAddress      string    `json:"ip_address"`
	Location       string    `json:"location"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivity   time.Time `json:"last_activity"`
	ExpiresAt      time.Time `json:"expires_at"`
	CurrentSession bool      `json:"current_session"`
}

func (uc *UserController) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  http.StatusUnauthorized,
			"message": "Not authenticated",
			"error":   "User not found in context",
		})
		return
	}

	var user models.User
	if err := uc.db.Select("id, email, username, full_name, created_at, last_login, failed_login_attempts").
		First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  http.StatusNotFound,
				"message": "User not found",
				"error":   "User does not exist",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  http.StatusInternalServerError,
			"message": "Failed to fetch user",
			"error":   "Database error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  http.StatusOK,
		"message": "User details retrieved",
		"data": map[string]interface{}{
			"user": map[string]interface{}{
				"id":        user.ID,
				"email":     user.Email,
				"username":  user.Username,
				"full_name": user.FullName,
			},
		},
	})
}

func (uc *UserController) GetActiveSessions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  http.StatusUnauthorized,
			"message": "Not authenticated",
			"error":   "User not found in context",
		})
		return
	}

	currentSessionID, _ := c.Get("sessionID")

	var sessions []models.UserSession
	if err := uc.db.Where("user_id = ? AND is_active = ? AND expires_at > ?",
		userID, true, time.Now()).
		Order("last_activity DESC").
		Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  http.StatusInternalServerError,
			"message": "Failed to fetch sessions",
			"error":   "Database error",
		})
		return
	}

	var sessionResponses []SessionResponse
	for _, session := range sessions {
		sessionResponses = append(sessionResponses, SessionResponse{
			ID:             session.ID,
			DeviceInfo:     session.DeviceInfo,
			IPAddress:      session.IPAddress,
			Location:       session.Location,
			CreatedAt:      session.CreatedAt,
			LastActivity:   session.LastActivity,
			ExpiresAt:      session.ExpiresAt,
			CurrentSession: session.ID == currentSessionID,
		})
	}

	var totalActiveSessions int64
	uc.db.Model(&models.UserSession{}).
		Where("user_id = ? AND is_active = ? AND expires_at > ?",
			userID, true, time.Now()).
		Count(&totalActiveSessions)

	response := map[string]interface{}{
		"sessions":              sessionResponses,
		"total_active_sessions": totalActiveSessions,
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  http.StatusOK,
		"message": "Active sessions retrieved successfully",
		"data":    response,
	})
}
