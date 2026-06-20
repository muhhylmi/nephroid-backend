# NephroAid Backend

This is the backend API for NephroAid, built using Golang (Go 1.22+) and PostgreSQL. 

It handles the core CRUD operations for Users, Chat Sessions, and Messages.

## Requirements
- Go 1.22 or higher
- Docker & Docker Compose (for the PostgreSQL database)

## Environment Configuration

1. Copy the provided `.env` file or ensure your environment variables are set.
2. The default configuration connects to the local Docker database instance.

## Running the Database

We use `docker-compose` to run the PostgreSQL database locally. From the root of the project:

```bash
cd ..
docker-compose up -d
```
*Note: The `db/init.sql` script will automatically create the required schema tables (Users, Chat Sessions, Messages) when the database starts for the first time.*

## Running the Server

Navigate into the `backend` directory, install dependencies, and run the server:

```bash
go mod tidy
go run ./cmd/api
```

The server will start listening on port `8080` (or the port defined in `.env`).

## API Endpoints

| Method | Endpoint | Description |
| --- | --- | --- |
| POST | `/api/users` | Register a new user |
| GET | `/api/users/{id}` | Get user by ID |
| POST | `/api/sessions` | Create a new chat session |
| GET | `/api/users/{userId}/sessions` | List all sessions for a user |
| DELETE | `/api/sessions/{id}` | Delete a chat session |
| POST | `/api/sessions/{id}/messages` | Add a message to a session |
| GET | `/api/sessions/{id}/messages` | List all messages in a session |

## Architecture Notes
- **Router**: Native standard library `net/http` `ServeMux` with wildcard path parameter support.
- **Database**: `pgxpool` for high-performance PostgreSQL connection pooling.
- **Graceful Shutdown**: The server properly listens for SIGTERM/SIGINT and finishes active requests before terminating.
- **Error Handling**: Follows single-handling rule (logs technical details at boundaries while returning sanitized HTTP error codes).
