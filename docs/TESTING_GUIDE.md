# Testing Guide

## Overview

This guide covers testing Disco on different platforms and configurations.

## Prerequisites

- Linux system (for full testing including NSS module)
- Go 1.21+
- GCC (for NSS module)
- Docker (optional, for multi-node testing)

## Quick Validation Tests

### 1. Configuration Validation

Test your configuration file before starting the daemon:

```bash
./disco config validate /path/to/config.yaml
```

This will:
- Validate YAML syntax
- Check all configuration values
- Show a summary of the configuration
- Report any errors

### 2. Build Verification

Ensure all components build successfully:

```bash
make clean
make all
```

This builds:
- disco-daemon (main daemon)
- disco (unified CLI tool)
- libnss_disco.so.2 (NSS module)

### 3. Basic Daemon Test

Start the daemon and verify it's running:

```bash
# Start daemon
sudo ./disco-daemon --config config.yaml

# In another terminal, test socket
ls -l /run/disco.sock

# Query the daemon
./disco hosts
```

## Multi-Node Testing

### Docker Testing (Linux Only)

Use the host networking Docker setup for broadcast support:

```bash
# Start 3 nodes
docker-compose -f docker-compose-host.yml up -d

# Wait for discovery (30-60 seconds)
sleep 60

# Check what web1 sees
docker exec -it disco-web1 disco hosts

# Check services
docker exec -it disco-web1 disco services

# View logs
docker-compose -f docker-compose-host.yml logs -f

# Stop
docker-compose -f docker-compose-host.yml down
```

### Manual Multi-Host Testing

For testing on actual hardware:

1. **Setup on each host:**
   ```bash
   sudo ./install.sh
   sudo systemctl start disco
   ```

2. **Configure each host:**
   ```bash
   # Edit /etc/disco/config.yaml
   # Set unique hostname
   # Configure same broadcast address
   ```

3. **Restart on each host:**
   ```bash
   sudo systemctl restart disco
   ```

4. **Test discovery (on any host):**
   ```bash
   disco hosts
   disco services
   ```

5. **Verify each host sees the others:**
   ```bash
   # On host1
   disco hosts | grep host2
   disco hosts | grep host3

   # On host2
   disco hosts | grep host1
   disco hosts | grep host3
   ```

## Automated Tests

### Run Test Script

The `test/quick-test.sh` script performs basic validation:

```bash
./test/quick-test.sh
```

This checks:
- Daemon is running
- Socket is accessible
- Query commands work
- Self-discovery works
- Services are detected

### Unit Tests

Run the Go unit tests:

```bash
make test
```

## Integration Testing

### NSS Module Testing

Test the NSS module on Linux:

```bash
# Install NSS module
sudo make install-libdisco only

# Configure nsswitch.conf
sudo sed -i 's/hosts: files dns/hosts: files disco dns/' /etc/nsswitch.conf

# Test name resolution
getent hosts web1
getent hosts mail1

# Verify it uses the daemon
grep web1 /var/log/disco.log
```

### Service Discovery Testing

Verify services are detected:

```bash
# Start a test service
python3 -m http.server 8080 &

# Wait for scan interval (default 60s)
sleep 65

# Check if service detected
disco services

# Should show www service on your host
```

### Security Testing (Optional)

Test security features:

```bash
# Generate keys
sudo disco key generate

# Add trusted peer
sudo disco key add-trusted <public-key>

# Enable security in config
# Edit /etc/disco/config.yaml
# Set: security.enabled: true
# Set: security.require_signed: true

# Restart daemon
sudo systemctl restart disco

# Verify signing works (check logs)
sudo journalctl -u disco -f
```

## Troubleshooting

### No Hosts Discovered

**Possible causes:**
1. Firewall blocking UDP port 5354
2. Different broadcast addresses
3. Discovery disabled in config

**Solutions:**
```bash
# Check firewall
sudo iptables -L | grep 5354
sudo firewall-cmd --list-ports | grep 5354

# Check config
disco config validate /etc/disco/config.yaml

# Enable discovery in config
# discovery.enabled: true
```

### Services Not Detected

**Possible causes:**
1. Service not on configured ports
2. Scan interval too long
3. Service detection disabled

**Solutions:**
```bash
# Check running services
sudo netstat -tlnp

# Lower scan interval for testing
# In config: discovery.scan_interval: 10s

# Restart daemon
sudo systemctl restart disco
```

### NSS Module Not Working

**Possible causes:**
1. NSS module not installed
2. nsswitch.conf not configured
3. Daemon not running
4. Library path issues

**Solutions:**
```bash
# Check NSS module installed
ls -la /lib/x86_64-linux-gnu/libnss_disco.so*

# Check nsswitch.conf
grep disco /etc/nsswitch.conf

# Check library loaded
ldconfig -p | grep nss_disco

# Test with getent
getent hosts localhost
```

### Docker Broadcast Not Working

**Issue:** Bridge networking doesn't support broadcast

**Solution:** Use host networking
```bash
# Use docker-compose-host.yml
docker-compose -f docker-compose.host.yml up -d
```

## Performance Testing

### Load Testing

Test with multiple hosts:

```bash
# Start daemon
sudo ./disco-daemon --config config.yaml &

# Generate test traffic
for i in {1..100}; do
    ./disco lookup host$i
done

# Check metrics (if available)
disco hosts | wc -l
```

### Memory Usage

Monitor daemon memory usage:

```bash
# Start daemon
sudo ./disco-daemon --config config.yaml &

# Monitor memory
watch -n 1 'ps aux | grep disco-daemon'
```

Expected: <20MB for typical deployment (50+ hosts)

## Log Analysis

### Debug Mode

Enable debug logging for troubleshooting:

```yaml
# In config.yaml
logging:
  level: "debug"
  format: "text"
```

### Key Log Messages

Watch for these log entries:

**Discovery:**
```
"Announcer started"
"Listener started"
"Discovery message from: hostname"
```

**Services:**
```
"Service detected: www on 80"
"Service announcement updated"
```

**Errors:**
```
"Failed to bind to broadcast address"
"Rate limit exceeded"
"Invalid signature"
```

## Test Checklist

- [ ] Configuration validates
- [ ] Daemon starts successfully
- [ ] Socket is created
- [ ] disco hosts works
- [ ] disco services works
- [ ] Self-host is discovered
- [ ] Other hosts are discovered (multi-node)
- [ ] Services are detected
- [ ] Name resolution works (NSS module)
- [ ] Daemon survives restart
- [ ] Graceful shutdown works

## CI/CD Integration

### Automated Tests

For CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Build
  run: make all

- name: Validate Config
  run: ./disco config validate config.yaml

- name: Unit Tests
  run: make test

- name: Integration Test
  run: |
    docker-compose -f docker-compose.host.yml up -d
    sleep 60
    docker exec disco-web1 disco hosts
    docker-compose down
```

## Next Steps

After passing tests:
1. Deploy to production
2. Monitor with journalctl
3. Use disco for diagnostics
4. Adjust configuration based on network size
5. Consider enabling security for production
