# RSVP Backend – Wedding Invitation & Media API

This is a Go-based backend API built for managing wedding invitations and guest logistics for a "Save the Date" project. It supports guest registration, seating assignments, ticket generation, and a collaborative photo gallery where guests can upload pictures of the wedding.

---

## ✨ Features

- RSVP management (confirm attendance)
- Guest table assignments
- Ticket generation per guest
- Guest photo uploads (wedding gallery)
- PostgreSQL migrations via CLI

---

## 🛠 Tech Stack

- **Language**: Go
- **Database**: PostgreSQL (via Supabase)

- **Migrations**: [`golang-migrate`](https://github.com/golang-migrate/migrate)
- **Environment Configuration**: `.env` file

---

## 🚀 Getting Started

### 1. Clone the repo

```bash

git clone https://github.com/DiegoB0/rsvp_server.git
cd rsvp_backend

```

### 2. Setup your .env file

Create a .env file in the root directory with the following structure

```bash
DB_PORT=
DB_HOST=
DB_USER=
DB_PASSWORD=
DB_NAME=
PORT=
```

### 3. Install dependencies and tools

Make sure Go is installed. Then install migrate if you haven't already:

```bash

brew install golang-migrate
```

### Makefile commands

| Command                                      | Description                                                                  |
| -------------------------------------------- | ---------------------------------------------------------------------------- |
| `make build`                                 | Build the project                                                            |
| `make run`                                   | Build and run the binary                                                     |
| `make test`                                  | Run Go tests                                                                 |
| `make migrate-up`                            | Apply all pending database migrations                                        |
| `make migrate-down`                          | Rollback the last migration                                                  |
| `make migrate-create name=create_table_name` | Create a new migration                                                       |
| `make docker-build`                          | Build the Docker image                                                       |
| `make docker-up`                             | Run the Docker container in detached mode                                    |
| `make docker-run`                            | Run the Docker container in foreground mode                                  |
| `make docker-stop`                           | Stop the running Docker container                                            |
| `make docker-rm`                             | Remove the stopped Docker container                                          |
| `make docker-clean`                          | Stop, remove container, and delete the Docker image                          |
| `make docker-dev`                            | Build the Docker image and run the container (shortcut for dev workflow)     |
| `make compose-up`                            | Build and start services using `docker-compose`                              |
| `make compose-down`                          | Stop and remove services and networks created by `docker-compose`            |
| `make compose-dev`                           | Build Docker image and run with `docker-compose` (preferred for development) |

### Running the server

```bash
make run
```

### Creating migrations

Migrations are located in `cmd/migrate/migrations`.

### 1. Create a new migration

```bash
make migrate-create name=create_users_table
```

This will generate something like this:

```bash
cmd/migrate/migrations/
  ├── 20250529220000_create_users_table.up.sql
  └── 20250529220000_create_users_table.down.sql

```

### 2. Create the sql

Once you've created the migrations just create the table with sql.

Example: `create_users_table.up.sql`:

```bash
CREATE TABLE IF NOT EXISTS users (

  id SERIAL PRIMARY KEY,
  name VARCHAR(250) NOT NULL,
  email VARCHAR(250) UNIQUE NOT NULL,
  password VARCHAR(250) NOT NULL,
);

```

Example: `create_users_table.down.sql`:

```bash

DROP TABLE IF EXISTS users;


```

### 3. Run the migration

```bash
make migrate-up
```

## 🐳 Docker integration

The project supports a docker-based development setup using make.

### ✅ Requirements

Before running the project, make sure you have the following installed:

- [Docker](https://www.docker.com/) – Containerization platform
- [Make](https://www.gnu.org/software/make/) – Tool to run predefined scripts easily

```bash
make docker-dev
```

## 📄 API Documentation

Once the project is running, you can access the Swagger UI at:

```bash
http://localhost:PORT/swagger/index.html
```

Replace PORT with the one your server is running on (e.g., 8080).
All API documentation is available there.
