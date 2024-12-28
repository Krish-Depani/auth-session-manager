package controllers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Krish-Depani/auth-session-manager/database"
	"github.com/Krish-Depani/auth-session-manager/models"
	"github.com/Krish-Depani/auth-session-manager/utils"
	"github.com/Krish-Depani/auth-session-manager/validators"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct {
	db    *gorm.DB
	redis *database.RedisClient
}

type AuthResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func NewAuthController(db *gorm.DB, redis *database.RedisClient) *AuthController {
	return &AuthController{
		db:    db,
		redis: redis,
	}
}

func (ac *AuthController) sendResponse(c *gin.Context, status int, message string, data interface{}, err interface{}) {
	c.JSON(status, AuthResponse{
		Status:  status,
		Message: message,
		Data:    data,
		Error:   err,
	})
}

func (ac *AuthController) Register(c *gin.Context) {
	req, ok := validators.ValidateRegisterRequest(c)
	if !ok {
		return
	}

	tx := ac.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var existingUser models.User
	if err := tx.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusConflict, "Registration failed", nil, map[string]string{
			"field":   "email_or_username",
			"message": "A user with this email or username already exists",
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Internal server error", nil, "Database error")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Registration failed", nil, "Failed to process password")
		return
	}

	user := models.User{
		Email:        strings.ToLower(req.Email),
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		IsActive:     true,
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Registration failed", nil, "Failed to create user")
		return
	}

	if err := tx.Commit().Error; err != nil {
		ac.sendResponse(c, http.StatusInternalServerError, "Registration failed", nil, "Failed to commit transaction")
		return
	}

	ac.sendResponse(c, http.StatusCreated, "User registered successfully", map[string]interface{}{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
	}, nil)
}

func (ac *AuthController) Login(c *gin.Context) {
	req, ok := validators.ValidateLoginRequest(c)
	if !ok {
		return
	}

	tx := ac.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var user models.User
	if err := tx.Where("email = ? AND is_active = ?", strings.ToLower(req.Email), true).First(&user).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ac.sendResponse(c, http.StatusUnauthorized, "Login failed", nil, map[string]string{
				"field":   "email",
				"message": "Invalid credentials",
			})
			return
		}
		ac.sendResponse(c, http.StatusInternalServerError, "Login failed", nil, "Database error")
		return
	}

	if user.FailedLoginAttempts >= 5 && user.LastFailedAttempt != nil {
		cooldownPeriod := time.Now().Add(-15 * time.Minute)
		if user.LastFailedAttempt.After(cooldownPeriod) {
			tx.Rollback()
			ac.sendResponse(c, http.StatusTooManyRequests, "Login failed", nil, map[string]string{
				"message":  "Too many failed attempts. Please try again later.",
				"cooldown": "15 minutes",
			})
			return
		}
		user.FailedLoginAttempts = 0
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		now := time.Now()
		if err := tx.Model(&user).Updates(map[string]interface{}{
			"failed_login_attempts": user.FailedLoginAttempts + 1,
			"last_failed_attempt":   now,
		}).Error; err != nil {
			tx.Rollback()
			ac.sendResponse(c, http.StatusInternalServerError, "Login failed", nil, "Failed to update login attempts")
			return
		}
		tx.Commit()
		ac.sendResponse(c, http.StatusUnauthorized, "Login failed", nil, map[string]string{
			"field":   "password",
			"message": "Invalid credentials",
		})
		return
	}

	sessionToken := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	location := utils.GetIPLocation(c.ClientIP())

	session := models.UserSession{
		UserID:       user.ID,
		SessionToken: sessionToken,
		DeviceInfo:   c.GetHeader("User-Agent"),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Location:     location,
		LastActivity: time.Now(),
		ExpiresAt:    expiresAt,
		IsActive:     true,
	}

	if err := tx.Create(&session).Error; err != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Login failed", nil, "Failed to create session")
		return
	}

	now := time.Now()
	if err := tx.Model(&user).Updates(map[string]interface{}{
		"last_login":            now,
		"failed_login_attempts": 0,
		"last_failed_attempt":   nil,
	}).Error; err != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Login failed", nil, "Failed to update user")
		return
	}

	err := ac.redis.SetSession(context.Background(), sessionToken, user.ID, 24*time.Hour)
	if err != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Login failed", nil, "Failed to create session")
		return
	}

	if err := tx.Commit().Error; err != nil {
		ac.sendResponse(c, http.StatusInternalServerError, "Login failed", nil, "Failed to commit transaction")
		return
	}

	c.SetCookie("session_token", sessionToken, int(24*time.Hour.Seconds()), "/", "", false, true)

	ac.sendResponse(c, http.StatusOK, "Login successful", map[string]interface{}{
		"user": map[string]interface{}{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
		},
	}, nil)
}

func (ac *AuthController) Logout(c *gin.Context) {
	sessionToken, err := c.Cookie("session_token")
	if err != nil {
		ac.sendResponse(c, http.StatusBadRequest, "Logout failed", nil, "No session found")
		return
	}

	tx := ac.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	result := tx.Model(&models.UserSession{}).
		Where("session_token = ? AND is_active = ?", sessionToken, true).
		Updates(map[string]interface{}{
			"is_active":  false,
			"expires_at": time.Now(),
		})

	if result.Error != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Logout failed", nil, "Failed to end session")
		return
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		ac.sendResponse(c, http.StatusBadRequest, "Logout failed", nil, "Invalid session")
		return
	}

	if err := ac.redis.DeleteSession(context.Background(), sessionToken); err != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Logout failed", nil, "Failed to delete session")
		return
	}

	if err := tx.Commit().Error; err != nil {
		ac.sendResponse(c, http.StatusInternalServerError, "Logout failed", nil, "Failed to commit transaction")
		return
	}

	c.SetCookie("session_token", "", -1, "/", "", false, true)

	ac.sendResponse(c, http.StatusOK, "Logged out successfully", nil, nil)
}

func (ac *AuthController) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, err := c.Cookie("session_token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, AuthResponse{
				Status:  http.StatusUnauthorized,
				Message: "Authentication required",
				Error:   "No session found",
			})
			return
		}

		userID, err := ac.redis.GetSession(c.Request.Context(), sessionToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, AuthResponse{
				Status:  http.StatusUnauthorized,
				Message: "Authentication failed",
				Error:   "Invalid session",
			})
			return
		}

		var session models.UserSession
		if err := ac.db.Where("session_token = ? AND is_active = ? AND expires_at > ?",
			sessionToken, true, time.Now()).First(&session).Error; err != nil {
			// Clean up Redis if session is invalid
			ac.redis.DeleteSession(c.Request.Context(), sessionToken)
			c.AbortWithStatusJSON(http.StatusUnauthorized, AuthResponse{
				Status:  http.StatusUnauthorized,
				Message: "Authentication failed",
				Error:   "Invalid or expired session",
			})
			return
		}

		ac.db.Model(&session).Update("last_activity", time.Now())

		c.Set("userID", userID)
		c.Set("sessionID", session.ID)

		c.Next()
	}
}
