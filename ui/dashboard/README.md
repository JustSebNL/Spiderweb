# Spiderweb Dashboard and Operator UI

This directory contains the web-based dashboard and operator interface for Spiderweb, providing real-time monitoring, management, and control capabilities.

## Overview

The dashboard provides a comprehensive interface for:
- **Real-time Monitoring**: View system health, performance metrics, and active operations
- **Agent Management**: Monitor and control agent instances and their configurations
- **Task Management**: View task queues, execution status, and results
- **System Configuration**: Modify runtime settings and configurations
- **Event Logs**: Access detailed logs and system events
- **Resource Management**: Monitor resource usage and system capacity

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Client    │    │   API Gateway   │    │  Operator Core  │
│                 │◄──►│                 │◄──►│                 │
│ • React/Vue.js  │    │ • WebSocket     │    │ • Event Stream  │
│ • Charts/Graphs │    │ • REST API      │    │ • Config Mgmt   │
│ • Real-time UI  │    │ • Auth/Security │    │ • Task Control  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │  Spiderweb Core │
                       │                 │
                       │ • Agent Manager │
                       │ • Task Queue    │
                       │ • Event Bus     │
                       │ • Health Check  │
                       └─────────────────┘
```

## Features

### 1. System Overview
- **Health Status**: Real-time system health indicators
- **Performance Metrics**: CPU, memory, network usage
- **Active Agents**: List of running agent instances
- **Task Queue**: Pending and completed tasks
- **Error Summary**: Recent errors and warnings

### 2. Agent Management
- **Agent List**: View all configured agents
- **Agent Status**: Real-time status and health
- **Configuration**: Modify agent settings
- **Logs**: Access agent-specific logs
- **Control**: Start/stop/restart agents

### 3. Task Monitoring
- **Task Queue**: View pending tasks with priority
- **Execution History**: Completed tasks with results
- **Task Details**: Detailed task information and logs
- **Performance**: Task execution times and success rates
- **Filtering**: Filter by agent, status, time range

### 4. Event System
- **Event Stream**: Real-time event feed
- **Event Filtering**: Filter by type, source, severity
- **Event Details**: Detailed event information
- **Alerts**: Configurable alert thresholds
- **Notifications**: Push notifications for critical events

### 5. Configuration Management
- **Runtime Config**: Modify system settings
- **Agent Config**: Update agent configurations
- **Feature Flags**: Enable/disable features
- **Environment Variables**: View and modify environment settings
- **Validation**: Configuration validation and testing

### 6. Resource Monitoring
- **System Resources**: CPU, memory, disk usage
- **Network**: Bandwidth and connection monitoring
- **Database**: Connection status and performance
- **External Services**: Status of external integrations
- **Capacity Planning**: Resource usage trends

## Installation

### Prerequisites
- Node.js 18+ 
- npm or yarn
- Spiderweb backend running

### Setup

1. **Install Dependencies**
   ```bash
   cd ui/dashboard
   npm install
   ```

2. **Configuration**
   ```bash
   cp .env.example .env
   # Edit .env with your backend API endpoint
   ```

3. **Build**
   ```bash
   npm run build
   ```

4. **Run Development Server**
   ```bash
   npm run dev
   ```

5. **Access Dashboard**
   Open browser to `http://localhost:3000`

## Configuration

### Environment Variables

```bash
# Backend API Configuration
VITE_API_BASE_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/ws

# Authentication
VITE_AUTH_ENABLED=true
VITE_AUTH_TYPE=basic  # or jwt, oauth

# Features
VITE_REAL_TIME_ENABLED=true
VITE_METRICS_ENABLED=true
VITE_LOGS_ENABLED=true

# UI Settings
VITE_THEME=dark
VITE_REFRESH_INTERVAL=5000  # ms
```

### Backend Integration

The dashboard integrates with Spiderweb through:

1. **REST API**: For configuration and control operations
2. **WebSocket**: For real-time event streaming
3. **Event Bus**: For system notifications
4. **Health Checks**: For system status monitoring

## API Endpoints

### System Status
- `GET /api/v1/status` - System health and status
- `GET /api/v1/metrics` - Performance metrics
- `GET /api/v1/health` - Component health checks

### Agent Management
- `GET /api/v1/agents` - List all agents
- `GET /api/v1/agents/{id}` - Get agent details
- `POST /api/v1/agents/{id}/start` - Start agent
- `POST /api/v1/agents/{id}/stop` - Stop agent
- `PUT /api/v1/agents/{id}/config` - Update configuration

### Task Management
- `GET /api/v1/tasks` - List tasks
- `GET /api/v1/tasks/{id}` - Get task details
- `POST /api/v1/tasks` - Create new task
- `DELETE /api/v1/tasks/{id}` - Cancel task

### Event Stream
- `GET /api/v1/events` - Event history
- `WS /ws/events` - Real-time event stream
- `POST /api/v1/events/subscribe` - Subscribe to events

### Configuration
- `GET /api/v1/config` - Get configuration
- `PUT /api/v1/config` - Update configuration
- `POST /api/v1/config/validate` - Validate configuration

## WebSocket Events

The dashboard uses WebSocket for real-time updates:

### System Events
```javascript
// Agent status update
{ type: 'agent_status', agentId: 'agent-1', status: 'running' }

// Task completion
{ type: 'task_completed', taskId: 'task-123', result: {...} }

// System health
{ type: 'system_health', status: 'healthy', metrics: {...} }
```

### Error Events
```javascript
// Agent error
{ type: 'agent_error', agentId: 'agent-1', error: 'Connection failed' }

// Task failure
{ type: 'task_failed', taskId: 'task-123', error: 'Timeout' }
```

## Security

### Authentication
- JWT token-based authentication
- Role-based access control (RBAC)
- Session management
- CSRF protection

### Authorization
- Permission-based access to features
- Agent-specific access control
- Configuration change approval workflow

### Security Headers
- Content Security Policy (CSP)
- HTTPS enforcement
- Secure cookies
- Rate limiting

## Deployment

### Docker
```bash
# Build Docker image
docker build -t spiderweb-dashboard .

# Run container
docker run -p 3000:3000 spiderweb-dashboard
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spiderweb-dashboard
spec:
  replicas: 2
  selector:
    matchLabels:
      app: spiderweb-dashboard
  template:
    metadata:
      labels:
        app: spiderweb-dashboard
    spec:
      containers:
      - name: dashboard
        image: spiderweb-dashboard:latest
        ports:
        - containerPort: 3000
        env:
        - name: API_BASE_URL
          value: "http://spiderweb-api:8080"
```

### Production Build
```bash
# Build for production
npm run build

# Serve with nginx
nginx -s reload
```

## Monitoring and Logging

### Client-side Logging
- Error tracking and reporting
- Performance monitoring
- User interaction logging
- Debug mode support

### Server-side Integration
- Log aggregation
- Performance metrics
- Error correlation
- Audit trails

## Development

### Project Structure
```
ui/dashboard/
├── src/
│   ├── components/     # React/Vue components
│   ├── pages/         # Page components
│   ├── services/      # API services
│   ├── store/         # State management
│   ├── utils/         # Utility functions
│   └── styles/        # CSS/SCSS styles
├── public/            # Static assets
├── tests/             # Test files
└── config/            # Build configuration
```

### Adding New Features
1. Create component in `src/components/`
2. Add page in `src/pages/`
3. Create service in `src/services/`
4. Update routing in `src/App.js`
5. Add tests in `tests/`

### Testing
```bash
# Run unit tests
npm test

# Run E2E tests
npm run test:e2e

# Run performance tests
npm run test:performance
```

## Troubleshooting

### Common Issues

1. **WebSocket Connection Failed**
   - Check backend WebSocket endpoint
   - Verify CORS settings
   - Check authentication

2. **API Requests Failing**
   - Verify API base URL
   - Check authentication tokens
   - Review browser console for errors

3. **Real-time Updates Not Working**
   - Check WebSocket connection
   - Verify event subscription
   - Check network connectivity

### Debug Mode
```bash
# Enable debug logging
localStorage.setItem('debug', 'spiderweb:*')

# View WebSocket events
localStorage.setItem('debug_ws', 'true')
```

## Contributing

1. Fork the repository
2. Create feature branch
3. Make changes
4. Add tests
5. Submit pull request

## Support

- Documentation: [docs/](../docs/)
- Issues: [GitHub Issues](https://github.com/JustSebNL/Spiderweb/issues)
- Community: [Discord](https://discord.gg/spiderweb)
- Email: support@spiderweb.ai

## License

This dashboard is part of the Spiderweb project and is licensed under the MIT License.