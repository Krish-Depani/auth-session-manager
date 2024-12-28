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

// sendResponse is a helper function to send consistent JSON responses
func (ac *AuthController) sendResponse(c *gin.Context, status int, message string, data interface{}, err interface{}) {
	c.JSON(status, AuthResponse{
		Status:  status,
		Message: message,
		Data:    data,
		Error:   err,
	})
}

// Register handles user registration
func (ac *AuthController) Register(c *gin.Context) {
	req, ok := validators.ValidateRegisterRequest(c)
	if !ok {
		return
	}

	// Start database transaction
	tx := ac.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if user exists
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

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Registration failed", nil, "Failed to process password")
		return
	}

	// Create user
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

	// Commit transaction
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

// Login handles user authentication
func (ac *AuthController) Login(c *gin.Context) {
	req, ok := validators.ValidateLoginRequest(c)
	if !ok {
		return
	}

	// Start transaction
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

	// Check for too many failed attempts
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
		// Reset counter after cooldown
		user.FailedLoginAttempts = 0
	}

	// Verify password
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

	// Generate session
	sessionToken := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	location := utils.GetIPLocation(c.ClientIP())

	// Create session in database
	session := models.UserSession{
		UserID:       user.ID,
		SessionToken: sessionToken,
		DeviceInfo:   c.GetHeader("User-Agent"),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Location:     location, // Implement geolocation if needed
		LastActivity: time.Now(),
		ExpiresAt:    expiresAt,
		IsActive:     true,
	}

	if err := tx.Create(&session).Error; err != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Login failed", nil, "Failed to create session")
		return
	}

	// Update user's last login
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

	// Store session in Redis
	err := ac.redis.SetSession(context.Background(), sessionToken, user.ID, 24*time.Hour)
	if err != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Login failed", nil, "Failed to create session")
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		ac.sendResponse(c, http.StatusInternalServerError, "Login failed", nil, "Failed to commit transaction")
		return
	}

	// Set cookie
	c.SetCookie("session_token", sessionToken, int(24*time.Hour.Seconds()), "/", "", false, true)

	ac.sendResponse(c, http.StatusOK, "Login successful", map[string]interface{}{
		"user": map[string]interface{}{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
		},
		// "session": map[string]interface{}{
		// 	"token":     sessionToken,
		// 	"expiresAt": expiresAt,
		// },
	}, nil)
}

// Logout handles user logout
func (ac *AuthController) Logout(c *gin.Context) {
	sessionToken, err := c.Cookie("session_token")
	if err != nil {
		ac.sendResponse(c, http.StatusBadRequest, "Logout failed", nil, "No session found")
		return
	}

	// Start transaction
	tx := ac.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Deactivate session in database
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

	// Remove session from Redis
	if err := ac.redis.DeleteSession(context.Background(), sessionToken); err != nil {
		tx.Rollback()
		ac.sendResponse(c, http.StatusInternalServerError, "Logout failed", nil, "Failed to delete session")
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		ac.sendResponse(c, http.StatusInternalServerError, "Logout failed", nil, "Failed to commit transaction")
		return
	}

	// Clear cookie
	c.SetCookie("session_token", "", -1, "/", "", false, true)

	ac.sendResponse(c, http.StatusOK, "Logged out successfully", nil, nil)
}

// AuthMiddleware handles authentication for protected routes
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

		// First check Redis for quick validation
		userID, err := ac.redis.GetSession(c.Request.Context(), sessionToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, AuthResponse{
				Status:  http.StatusUnauthorized,
				Message: "Authentication failed",
				Error:   "Invalid session",
			})
			return
		}

		// Verify session in database and update last activity
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

		// Update last activity
		ac.db.Model(&session).Update("last_activity", time.Now())

		// Set user context
		c.Set("userID", userID)
		c.Set("sessionID", session.ID)

		c.Next()
	}
}

// GetCurrentUser retrieves the current authenticated user
func (ac *AuthController) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		ac.sendResponse(c, http.StatusUnauthorized, "Authentication required", nil, "No user found in context")
		return
	}

	var user models.User
	if err := ac.db.Select("id, email, username, full_name, created_at, last_login").
		First(&user, userID).Error; err != nil {
		ac.sendResponse(c, http.StatusInternalServerError, "Failed to retrieve user", nil, "Database error")
		return
	}

	ac.sendResponse(c, http.StatusOK, "User retrieved successfully", map[string]interface{}{
		"id":         user.ID,
		"email":      user.Email,
		"username":   user.Username,
		"full_name":  user.FullName,
		"created_at": user.CreatedAt,
		"last_login": user.LastLogin,
	}, nil)
}
