# Deployment Guide

This guide covers deployment options and production considerations for the Modbus MCP Server.

## Quick Start Deployment

### Using Go Directly
```bash
# Clone the repository
git clone https://github.com/devidasjadhav/go-mdbus-mcp.git
cd go-mdbus-mcp/sample

# Build the application
go build -o modbus-server main.go

# Run the server
./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002
```

### Using Docker
```dockerfile
# Dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o modbus-server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/modbus-server .
EXPOSE 8080
CMD ["./modbus-server"]
```

```bash
# Build and run with Docker
docker build -t modbus-mcp-server .
docker run -p 8080:8080 -e MODBUS_IP=192.168.1.22 -e MODBUS_PORT=5002 modbus-mcp-server
```

## Configuration

### Command Line Arguments
```bash
./modbus-server [options]

Options:
  --modbus-ip string     Modbus server IP address (default "192.168.1.22")
  --modbus-port int      Modbus server port (default 502)
  --help                 Show help message
```

### Environment Variables
```bash
# Set environment variables
export MODBUS_IP=192.168.1.100
export MODBUS_PORT=502

# Run with environment variables
./modbus-server
```

### Configuration File (Future Enhancement)
```yaml
# config.yaml
modbus:
  ip: "192.168.1.22"
  port: 5002
  timeout: 10s
  slave_id: 0

server:
  port: 8080
  host: "0.0.0.0"

logging:
  level: "info"
  format: "json"
```

## Production Deployment

### System Requirements
- **OS**: Linux, Windows, or macOS
- **CPU**: 1 core minimum, 2+ cores recommended
- **Memory**: 128MB minimum, 256MB recommended
- **Network**: Reliable network connection to Modbus devices

### Security Considerations

#### Network Security
```bash
# Run behind reverse proxy (nginx example)
server {
    listen 80;
    server_name your-domain.com;

    location /mcp {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### TLS/SSL Configuration
```bash
# Generate SSL certificate
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365

# Run with TLS (requires code modification)
./modbus-server --tls-cert cert.pem --tls-key key.pem
```

#### Access Control
- Implement authentication if needed
- Use network segmentation
- Restrict Modbus server access
- Monitor access logs

### Process Management

#### Systemd Service
```ini
# /etc/systemd/system/modbus-mcp.service
[Unit]
Description=Modbus MCP Server
After=network.target

[Service]
Type=simple
User=modbus
Group=modbus
WorkingDirectory=/opt/modbus-mcp
ExecStart=/opt/modbus-mcp/modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002
Restart=always
RestartSec=5
Environment=MODBUS_IP=192.168.1.22
Environment=MODBUS_PORT=5002

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start service
sudo systemctl enable modbus-mcp
sudo systemctl start modbus-mcp
sudo systemctl status modbus-mcp
```

#### Docker Compose
```yaml
# docker-compose.yml
version: '3.8'
services:
  modbus-mcp:
    build: .
    ports:
      - "8080:8080"
    environment:
      - MODBUS_IP=192.168.1.22
      - MODBUS_PORT=5002
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/mcp"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Monitoring and Logging

#### Log Management
```bash
# Log to file
./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002 > server.log 2>&1

# Use logrotate for log rotation
# /etc/logrotate.d/modbus-mcp
/var/log/modbus-mcp/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 644 modbus modbus
}
```

#### Health Checks
```bash
# Health check endpoint (requires implementation)
curl -f http://localhost:8080/health

# Docker health check
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
```

#### Monitoring Integration
```bash
# Prometheus metrics (future enhancement)
# Expose metrics endpoint
curl http://localhost:8080/metrics

# Grafana dashboard integration
# - Request latency
# - Error rates
# - Connection status
# - Modbus operation counts
```

## Scaling and Performance

### Single Instance Scaling
- **Concurrent Requests**: Supports multiple simultaneous operations
- **Memory Usage**: ~50-100MB per instance
- **CPU Usage**: Minimal for typical workloads
- **Network**: Depends on Modbus device response times

### Load Balancing
```nginx
# nginx load balancer configuration
upstream modbus_backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}

server {
    listen 80;
    location /mcp {
        proxy_pass http://modbus_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

### High Availability
```yaml
# Kubernetes deployment for HA
apiVersion: apps/v1
kind: Deployment
metadata:
  name: modbus-mcp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: modbus-mcp
  template:
    metadata:
      labels:
        app: modbus-mcp
    spec:
      containers:
      - name: modbus-mcp
        image: modbus-mcp:latest
        ports:
        - containerPort: 8080
        env:
        - name: MODBUS_IP
          value: "192.168.1.22"
        - name: MODBUS_PORT
          value: "5002"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Backup and Recovery

### Configuration Backup
```bash
# Backup configuration
cp config.yaml config.yaml.backup

# Backup with timestamp
cp config.yaml config.yaml.$(date +%Y%m%d_%H%M%S)
```

### Log Archiving
```bash
# Archive old logs
tar -czf logs-$(date +%Y%m%d).tar.gz /var/log/modbus-mcp/

# Clean old archives (keep last 30 days)
find /var/log/modbus-mcp/ -name "*.tar.gz" -mtime +30 -delete
```

## Troubleshooting Production Issues

### Common Issues

#### Connection Problems
```bash
# Check Modbus connectivity
telnet 192.168.1.22 5002

# Test with different timeout
./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002 --timeout 30

# Check network routing
traceroute 192.168.1.22
```

#### Performance Issues
```bash
# Monitor system resources
top -p $(pgrep modbus-server)

# Check network latency
ping 192.168.1.22

# Profile application
go tool pprof http://localhost:8080/debug/pprof/profile
```

#### Memory Leaks
```bash
# Monitor memory usage
ps aux | grep modbus-server

# Check for goroutine leaks
curl http://localhost:8080/debug/pprof/goroutine

# Restart service if needed
sudo systemctl restart modbus-mcp
```

### Debug Mode
```bash
# Enable debug logging
export LOG_LEVEL=debug
./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002

# Log to file with debug
./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002 2>&1 | tee debug.log
```

## Maintenance Tasks

### Regular Maintenance
```bash
# Update dependencies
go mod tidy
go mod download

# Rebuild application
go build -o modbus-server main.go

# Restart service
sudo systemctl restart modbus-mcp

# Check logs
sudo journalctl -u modbus-mcp -f
```

### Security Updates
```bash
# Update Go version
# Update dependencies
go get -u all
go mod tidy

# Rebuild and redeploy
go build -o modbus-server main.go
sudo systemctl restart modbus-mcp
```

### Database Maintenance (Future)
```bash
# Backup database
# Clean old data
# Optimize indexes
# Update schema
```

## Environment-Specific Configurations

### Development Environment
```bash
# Use local Modbus simulator
./modbus-server --modbus-ip 127.0.0.1 --modbus-port 502

# Enable debug logging
export LOG_LEVEL=debug

# Use development database
export DB_URL=postgres://localhost/dev_db
```

### Staging Environment
```bash
# Use staging Modbus server
./modbus-server --modbus-ip 192.168.2.22 --modbus-port 5002

# Enable info logging
export LOG_LEVEL=info

# Use staging database
export DB_URL=postgres://staging/db
```

### Production Environment
```bash
# Use production Modbus server
./modbus-server --modbus-ip 192.168.1.22 --modbus-port 5002

# Enable warn logging
export LOG_LEVEL=warn

# Use production database
export DB_URL=postgres://prod/db

# Enable TLS
./modbus-server --tls-cert /etc/ssl/certs/cert.pem --tls-key /etc/ssl/private/key.pem
```

## Disaster Recovery

### Backup Strategy
1. **Application Code**: Version controlled in Git
2. **Configuration**: Backup config files
3. **Logs**: Archive and rotate logs
4. **Database**: Regular backups (future feature)

### Recovery Procedures
1. **Application Failure**:
   ```bash
   sudo systemctl restart modbus-mcp
   ```

2. **Server Failure**:
   ```bash
   # Restore from backup
   # Rebuild application
   # Restore configuration
   # Start services
   ```

3. **Data Loss**:
   ```bash
   # Restore from backup
   # Verify data integrity
   # Update dependent systems
   ```

## Compliance and Auditing

### Security Compliance
- Regular security updates
- Access logging and monitoring
- Network segmentation
- Encryption for sensitive data

### Operational Compliance
- Service level agreements
- Incident response procedures
- Change management
- Documentation standards

### Audit Trail
- Access logs with timestamps
- Configuration change history
- System event logging
- Performance metrics

---

*This deployment guide should be reviewed and updated regularly to reflect changes in the deployment process and best practices.*