# Pactus Nodes Tracker Backend

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-13+-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)

A robust Go backend service for monitoring and tracking Pactus blockchain network nodes. This backend provides APIs for the Pactus Nodes Tracker frontend and handles real-time node health monitoring, data collection, and network analysis.

## 🎯 Project Overview

The **Pactus Nodes Tracker Backend** is part of a comprehensive multi-phased project designed to create a monitoring system for the geographically distributed Pactus blockchain network. This project is managed under the FUSION program with specific financial, legal, and quality standards established by the Pactus team.

### Core Objectives

1. **Bootstrap Node Health Monitoring**: Daily monitoring and scoring of critical bootstrap nodes
2. **Network Node Discovery**: Detection and tracking of all reachable Pactus nodes
3. **Public Node Management**: Registration and monitoring system for user-submitted nodes
4. **Data APIs**: Comprehensive APIs for external access to network data

## 📋 Project Scope & Requirements

### Phase-Based Development

#### Phase 1: Bootstrap Node Health ✅ (Current Implementation)
- **Daily Monitoring**: Automated daily health checks of bootstrap nodes
- **Connectivity Scoring**: 5-attempt connection testing with scoring algorithm
- **Health Classification**: Nodes classified as healthy (green) or unhealthy (red/gray)
- **Error Logging**: Comprehensive error tracking for investigation
- **Visual Indicators**: Data for frontend daily bar indicators

#### Phase 2: Reachable Nodes 🔄 (Planned)
- **Node Discovery**: Detection and display of all reachable network nodes
- **BitNodes Integration**: Reference implementation based on BitNodes project
- **Pagu API Integration**: Utilize Pagu project APIs for peer identification
- **Geographic Mapping**: Location data for world map visualization
- **Public Node Registration**: User registration system for JSON-RPC and gRPC nodes
- **Multi-page Support**: APIs for Bootstrap, JSON-RPC, and gRPC node pages

#### Phase 3: Node Crawler 📋 (Future)
- **Systematic Data Collection**: Advanced crawler using Nebula project reference
- **Enhanced Analytics**: Comprehensive network analysis and insights
- **Grafana Integration**: Optional advanced monitoring dashboards

#### Phase 4: Public APIs 📋 (Future)
- **JSON-RPC APIs**: Standardized public APIs for external access
- **Documentation**: Comprehensive API documentation
- **Community Access**: Open access for developers and researchers

## 🏗️ Technical Architecture

### Technology Stack

- **Language**: Go 1.21+
- **Database**: PostgreSQL 13+
- **Web Framework**: Gin (HTTP router and middleware)
- **Database Driver**: lib/pq (PostgreSQL driver)
- **Migrations**: golang-migrate
- **Scheduling**: robfig/cron (for periodic tasks)
- **Logging**: Logrus (structured logging)
- **Configuration**: godotenv (environment management)


## 🚀 Getting Started

### Prerequisites

- **Go**: Version 1.21 or higher
- **PostgreSQL**: Version 13 or higher
- **Git**: For version control

### Installation & Setup

1. **Clone the Repository**
   ```bash
   git clone https://github.com/kyvra-tech/pactus-nodes-tracker-backend.git
   cd pactus-nodes-tracker-backend
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Database Setup**
   ```bash
   # Create PostgreSQL database
   createdb pactus_tracker
   
   # Copy environment configuration
   cp .env.example .env
   
   # Edit .env with your database credentials
   nano .env
   ```

4. **Environment Configuration**
   ```env
   # Database Configuration
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=pactus_user
   DB_PASSWORD=pactus_password
   DB_NAME=pactus_tracker
   DB_SSLMODE=disable

   # Server Configuration
   SERVER_PORT=4622
   SERVER_HOST=0.0.0.0

   # Bootstrap Nodes Configuration
   BOOTSTRAP_CHECK_INTERVAL=24h
   CONNECTION_TIMEOUT=30s
   MAX_RETRY_ATTEMPTS=5

   # Logging
   LOG_LEVEL=info
   LOG_FORMAT=json
   ```

5. **Run Database Migrations**
   ```bash
   go run cmd/server/main.go migrate
   ```

6. **Start the Server**
   ```bash
   go run cmd/server/main.go
   ```

### Docker Development Environment

```bash
# Start PostgreSQL with Docker Compose
docker-compose up -d postgres

# Run the application
go run cmd/server/main.go
```

## 📊 Database Schema

### Bootstrap Nodes Table
```sql
CREATE TABLE bootstrap_nodes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    website VARCHAR(255),
    address TEXT NOT NULL UNIQUE,
    overall_score DECIMAL(5,2) DEFAULT 0.00,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Daily Status Table
```sql
CREATE TABLE daily_status (
    id SERIAL PRIMARY KEY,
    node_id INTEGER NOT NULL REFERENCES bootstrap_nodes(id),
    date DATE NOT NULL,
    color INTEGER NOT NULL CHECK (color IN (0, 1, 2)),
    attempts INTEGER DEFAULT 0,
    success BOOLEAN DEFAULT false,
    error_msg TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(node_id, date)
);
```

## 🔌 API Endpoints

### Phase 1 APIs (Current)

#### Get Bootstrap Nodes Health
```http
GET /api/v1/bootstrap
```

**Response:**
```json
[
  {
    "name": "Pactus",
    "email": "info@pactus.org",
    "website": "https://pactus.org",
    "address": "/dns/bootstrap1.pactus.org/tcp/21888/p2p/12D3KooWMnDsu8TDTk2VV8uD8zsNSB6eUkqtQs6ttg4bHq9zNaBe",
    "status": [
      {"color": 1, "date": "2024-01-15"},
      {"color": 0, "date": "2024-01-14"},
      {"color": 1, "date": "2024-01-13"}
    ],
    "overallScore": 85.5
  }
]
```

#### Health Check
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0"
}
```

### Planned APIs (Future Phases)

- `GET /api/v1/peers` - Peer nodes with geographic data
- `POST /api/v1/register` - Register public nodes
- `GET /api/v1/stats` - Network statistics
- `GET /api/v1/history` - Historical data

## ⚙️ Core Services

### Node Checker Service
- **Connection Testing**: TCP connection attempts with configurable timeout
- **Address Parsing**: Support for DNS and IP4/IP6 multiaddr formats
- **Retry Logic**: Configurable retry attempts with exponential backoff
- **Error Handling**: Comprehensive error logging and reporting

### Bootstrap Monitor Service
- **Daily Scheduling**: Automated daily health checks
- **Score Calculation**: 30-day rolling average health scoring
- **Status Tracking**: Historical status data with date-based records
- **Database Integration**: Efficient data storage and retrieval

### Scheduler Service
- **Cron Jobs**: Configurable scheduled task execution
- **Health Monitoring**: Daily bootstrap node checks
- **Error Recovery**: Robust error handling and retry mechanisms

## 🧪 Testing & Quality Assurance

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/services/...
```

### Code Quality
```bash
# Format code
go fmt ./...

# Lint code (requires golangci-lint)
golangci-lint run

# Vet code
go vet ./...
```

## 📈 Monitoring & Logging

### Structured Logging
- **Logrus Integration**: JSON-formatted structured logging
- **Log Levels**: Configurable logging levels (debug, info, warn, error)
- **Contextual Logging**: Request tracing and correlation IDs

### Health Monitoring
- **Health Endpoints**: Built-in health check endpoints
- **Metrics Collection**: Performance and operational metrics
- **Error Tracking**: Comprehensive error logging and alerting

## 🔒 Security & Configuration

### Security Features
- **Input Validation**: Comprehensive request validation
- **SQL Injection Prevention**: Parameterized queries
- **CORS Support**: Configurable cross-origin resource sharing
- **Rate Limiting**: API rate limiting and throttling

### Configuration Management
- **Environment Variables**: 12-factor app configuration
- **Default Values**: Sensible defaults for all settings
- **Validation**: Configuration validation on startup

## 🚀 Deployment

### Production Build
```bash
# Build binary
go build -o pactus-tracker cmd/server/main.go

# Run binary
./pactus-tracker
```

### Docker Deployment
```bash
# Build Docker image
docker build -t pactus-tracker-backend .

# Run container
docker run -p 4622:4622 pactus-tracker-backend
```

### Environment Setup
- **Database**: PostgreSQL with connection pooling
- **Reverse Proxy**: Nginx or similar for production
- **SSL/TLS**: HTTPS termination at load balancer
- **Monitoring**: Prometheus/Grafana for metrics

## 📝 Development Guidelines

### Code Standards
- **Go Conventions**: Follow standard Go coding conventions
- **Error Handling**: Comprehensive error handling and logging
- **Testing**: Unit tests for all business logic
- **Documentation**: Inline code documentation and README updates

### Git Workflow
- **Feature Branches**: Use feature branches for development
- **Pull Requests**: Code review required for all changes
- **Commit Messages**: Descriptive commit messages
- **Versioning**: Semantic versioning for releases

## 🤝 Contributing

### Development Process
1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Submit a pull request
5. Code review and approval

### Quality Requirements
- **Test Coverage**: Minimum 80% test coverage
- **Code Review**: All changes require review
- **Documentation**: Update documentation for new features
- **Linting**: Pass all linting checks

## 📄 License & Legal

### MIT License
This project is licensed under the MIT License as part of the FUSION program. This allows the Pactus team and community to use, modify, and distribute the work while the grantee retains authorship.

