# Auth Session Manager

A robust session-based authentication system built with Go, featuring Redis for session management and PostgreSQL for persistent data storage.

## Links

- [Working API's](https://www.postman.com/envolve-7536/workspace/auth-session-manager/collection/16623333-cbd26cf3-eea0-480f-b7fd-e50ab825496f?action=share&creator=16623333)
- [Docker Image](https://hub.docker.com/r/krishdepani/auth-session-manager/tags)

## Features

- Secure user authentication with session management
- Dual database system: PostgreSQL (persistent data) + Redis (session storage)
- Session tracking with device info and location
- Protection against brute force attacks
- Active session management and monitoring
- Automatic session expiration
- Secure password handling with bcrypt

## Prerequisites

- Go 1.x
- PostgreSQL
- Redis
- GNU Make (for using Makefile commands)

## Installation

1. Clone the repository:

```bash
git clone https://github.com/Krish-Depani/auth-session-manager.git
cd auth-session-manager
```

2. Install dependencies:

```bash
go mod download
```

3. Rename the `.env.example` file to `.env` and update the following environment variables as needed

4. Set up the databases:
   - Create PostgreSQL database
   - Start Redis server
   - Run migrations:

```bash
make migrate-up
```

## Running the Application

### Development Mode

```bash
make start-dev
```

### Production Mode

```bash
make start-prod
```

## API Routes

### Authentication Endpoints

- `POST /auth/register` - Register a new user
- `POST /auth/login` - User login
- `POST /auth/logout` - User logout (requires authentication)

### User Endpoints

- `GET /auth/user/me` - Get current user details (requires authentication)
- `GET /auth/user/sessions` - Get active sessions (requires authentication)

## Security Features

1. **Session Management**

   - Session tokens stored in Redis with TTL
   - Device and location tracking for each session
   - Active session monitoring

2. **Brute Force Protection**

   - Maximum login attempt limits
   - Cool-down period after failed attempts
   - Automatic account protection

3. **Secure Authentication**
   - Bcrypt password hashing
   - HTTP-only cookies for session tokens
   - Transaction-based operations for data consistency

## Database Management

### Creating New Migrations

```bash
make migrate-create name=migration_name
```

### Migration Commands

- Up: `make migrate-up`
- Down: `make migrate-down n=1`
- Status: `make migrate-status`
- Force Version: `make migrate-force version=1`

## Project Structure

```
├── bin/                  # Compiled binary
├── config/              # Configuration files
├── controllers/         # Request handlers
├── database/           # Database connections and migrations
├── models/             # Data models
├── routes/             # API route definitions
├── utils/              # Utility functions
└── validators/         # Request validation
```

## If you like this project, please give it a 🌟.

## Thank you 😊.
