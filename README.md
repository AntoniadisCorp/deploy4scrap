# Deploy Containers 4 Scrap

Deploy4Scrap is a Go Fiber API that automates deploying, scaling, and managing Fly.io machines using API requests. It supports Firebase JWT authentication and integrates with Prometheus for auto-scaling based on metrics.

## Features

- Clone or create new Fly.io machines
- Start, stop, and delete machines
- Execute tasks on running machines
- Firebase JWT authentication
- Prometheus metrics for fly auto-scaling

## Installation

1. **Clone the Repository**
   ```sh
   git clone https://github.com/AntoniadisCorp/deploy4scrap.git
   cd deploy4scrap
   ```
2. **Set Up Environment Variables**
   ```sh
   cp .env.txt .env
   ```
3. **Run the Application**
   ```sh
   go run main.go
   ```
4. **Deploy to Fly.io**
   ```sh
   flyctl deploy
   ```

## API Endpoints

| Method | Endpoint                | Description             |
| ------ | ----------------------- | ----------------------- |
| POST   | `/deploy`               | Deploy a new machine    |
|        | `clone=true&master_id=` | query                   |
| PUT    | `/machine/:id/start`    | Start a machine         |
| PUT    | `/machine/:id/stop`     | Stop a machine          |
| DELETE | `/machine/:id`          | Delete a machine        |
| POST   | `/execute-task/:id`     | Run a task on a machine |

---
