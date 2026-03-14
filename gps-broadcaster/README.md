# GPS Time Broadcasters for Disco Daemon

These standalone services broadcast GPS time via UDP for disco-daemon to receive.

## Protocol

All broadcasters send TIME_ANNOUNCE messages via UDP broadcast on port 5354:

```json
{
  "type": "TIME_ANNOUNCE",
  "message_id": "gps-source-1234567890",
  "timestamp": 1708123456789000000,
  "clock_info": {
    "stratum": 1,
    "precision": -20,
    "root_delay": 0.0,
    "root_dispersion": 0.0001,
    "reference_id": "GPS",
    "reference_time": 1708123456000000000
  },
  "leap_indicator": 0,
  "source_id": "gps-source"
}
```

## macOS / Linux (Go)

Built as `disco-gps-broadcaster` from the main project.

### Building

```bash
# Build all binaries including GPS broadcaster
make

# Or cross-compile for Pi
make cross-compile

# The binary will be at:
# - ./disco-gps-broadcaster (native)
# - dist/disco-gps-broadcaster-linux-arm (Pi Zero)
# - dist/disco-gps-broadcaster-linux-arm64 (Pi Zero 2W, Pi 4)
```

### Installation on Pi

```bash
# Copy to Pi
scp disco-gps-broadcaster pi@raspberrypi:/home/pi/
```

### Usage

```bash
# Basic usage
./disco-gps-broadcaster -device /dev/ttyACM0

# macOS
./disco-gps-broadcaster -device /dev/cu.usbmodem114401

# With options
./disco-gps-broadcaster \
  -device /dev/ttyACM0 \
  -broadcast 255.255.255.255:5354 \
  -id gps-pi-01 \
  -interval 16s \
  -interfaces eth0,wlan0 \
  -v
```

### systemd Service

Create `/etc/systemd/system/disco-gps-broadcaster.service`:

```ini
[Unit]
Description=Disco GPS Time Broadcaster
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/disco-gps-broadcaster -device /dev/ttyACM0 -v
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable:

```bash
sudo systemctl enable disco-gps-broadcaster
sudo systemctl start disco-gps-broadcaster
```

## Arduino

Location: `arduino/gps_time_broadcaster.ino`

### Hardware

- Arduino with Ethernet/WiFi shield
- GPS module (e.g., NEO-6M) connected via Serial

### Wiring

| GPS Module | Arduino |
|------------|---------|
| VCC        | 5V      |
| GND        | GND     |
| TX         | RX (pin 0 or Serial1) |
| RX         | TX (pin 1 or Serial1) |

### Build Options

**Arduino IDE:**
1. Open `gps_time_broadcaster.ino`
2. Select board and port
3. Upload

**PlatformIO:**
```bash
cd arduino
pio run -e arduino
pio run -e arduino --target upload
```

### Configuration

Edit the sketch to configure:

```cpp
// Network (choose one)
#define USE_ETHERNET 1
// #define USE_WIFI 1

// WiFi credentials (if using WiFi)
char ssid[] = "YOUR_SSID";
char pass[] = "YOUR_PASSWORD";

// Source ID
const char SOURCE_ID[] = "gps-arduino-01";

// Broadcast interval
const unsigned long BROADCAST_INTERVAL = 16000;
```

## ESPHome

Location: `esphome/`

### Files

- `gps-broadcaster.yaml` - Main configuration
- `gps_broadcaster.h` - Custom component

**Note:** The `gps_broadcaster.h` file is an ESPHome custom component. It will show errors in standard C++ IDEs because it depends on ESPHome's build system to provide headers like `esphome.h`. These errors are expected and do not indicate a problem - the file compiles correctly when built with ESPHome.

### Hardware

- ESP32 or ESP8266 board
- GPS module connected via UART

### Wiring (ESP32)

| GPS Module | ESP32 |
|------------|-------|
| VCC        | 3.3V  |
| GND        | GND   |
| TX         | GPIO16 (RX2) |
| RX         | GPIO17 (TX2) |

### Installation

1. Copy both files to your ESPHome config directory
2. Edit `gps-broadcaster.yaml`:
   - Update WiFi credentials
   - Change `source_id` if desired
3. Flash:

```bash
esphome run gps-broadcaster.yaml
```

### Configuration Options

In the YAML file:

```yaml
custom_component:
  - lambda: |-
      auto broadcaster = new GPSBroadcasterComponent();
      broadcaster->set_source_id("gps-esphome-01");  // Unique ID
      broadcaster->set_interval(16000);               // ms between broadcasts
      broadcaster->set_port(5354);                    // UDP port
      App.register_component(broadcaster);
      return {broadcaster};
```

## Testing

### Verify Broadcasts

On any machine with disco-daemon:

```bash
# Enable time sync in config.yaml
time_sync:
  enabled: true
  min_sources: 1  # For testing with single source
  require_signed: false

# Start daemon and check logs
./disco-daemon -config config.yaml

# Check time status
./disco-time
```

### Listen for Packets

```bash
# Using netcat
nc -ul 5354

# Using tcpdump
sudo tcpdump -i any udp port 5354 -A
```

## GPS Module Recommendations

- **u-blox NEO-6M / NEO-8M**: Common, affordable, good performance
- **u-blox NEO-M9N**: Better accuracy, faster lock
- **Quectel L86**: Compact, low power

For best time accuracy, use modules with PPS (Pulse Per Second) output. The current implementation uses NMEA time which has ~1 second accuracy. PPS-based solutions can achieve microsecond accuracy.

## Splitting Off

These broadcasters are designed to be standalone:

- **Arduino/ESPHome**: Copy the `arduino/` or `esphome/` directory - no dependencies on disco-daemon
- **Go broadcaster**: Built as part of disco-daemon but can be used independently - only depends on Go standard library
