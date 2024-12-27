# auth-session-manager

```sql
-- PostgreSQL Schema

-- Users table for storing user information
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    failed_login_attempts INT DEFAULT 0,
    last_failed_attempt TIMESTAMP WITH TIME ZONE
);

-- User sessions table for tracking active sessions
CREATE TABLE user_sessions (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) UNIQUE NOT NULL,
    device_info VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    location VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN DEFAULT true
);

-- Create indexes for better query performance
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX idx_users_email ON users(email);

-- Common Queries

-- 1. Create new user
INSERT INTO users (email, username, password_hash, full_name)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- 2. Get user by email (for login)
SELECT \* FROM users WHERE email = $1;

-- 3. Update failed login attempts
UPDATE users
SET failed_login_attempts = failed_login_attempts + 1,
last_failed_attempt = CURRENT_TIMESTAMP
WHERE email = $1;

-- 4. Reset failed login attempts after successful login
UPDATE users
SET failed_login_attempts = 0,
last_failed_attempt = NULL,
last_login = CURRENT_TIMESTAMP
WHERE id = $1;

-- 5. Create new session
INSERT INTO user_sessions
(user_id, session_token, device_info, ip_address, user_agent, location, expires_at)
VALUES
($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP + INTERVAL '24 hours')
RETURNING id;

-- 6. Get active sessions count for user (to check against MAX_DEVICES constant)
SELECT COUNT(\*)
FROM user_sessions
WHERE user_id = $1
AND is_active = true
AND expires_at > CURRENT_TIMESTAMP;

-- 7. Get all active sessions for user (for displaying device list)
SELECT us.\*, u.email
FROM user_sessions us
JOIN users u ON us.user_id = u.id
WHERE us.user_id = $1
AND us.is_active = true
AND us.expires_at > CURRENT_TIMESTAMP
ORDER BY us.last_activity DESC;

-- 8. Deactivate session (logout)
UPDATE user_sessions
SET is_active = false
WHERE session_token = $1;

-- 9. Clean up expired sessions
DELETE FROM user_sessions
WHERE expires_at < CURRENT_TIMESTAMP;

-- 10. Check if session is valid
SELECT us.\*, u.email, u.is_active as user_active
FROM user_sessions us
JOIN users u ON us.user_id = u.id
WHERE us.session_token = $1
AND us.is_active = true
AND us.expires_at > CURRENT_TIMESTAMP;

-- Redis Schema (Key-Value Pairs)

-- Session Token Storage
-- Key: "session:{session_token}"
-- Value: JSON containing:
-- {
-- "user_id": "123",
-- "expires_at": "2024-12-28T00:00:00Z",
-- "ip_address": "192.168.1.1",
-- "device_info": "Chrome on MacOS"
-- }
-- TTL: 24 hours

-- Rate Limiting
-- Key: "login_attempts:{ip_address}"
-- Value: Number of attempts
-- TTL: 15 minutes

-- User Sessions Count
-- Key: "user_sessions:{user_id}"
-- Value: Set of active session tokens
-- TTL: None (cleaned up on logout)
```
