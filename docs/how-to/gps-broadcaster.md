# GPS Broadcaster

The GPS broadcaster sends `TIME_ANNOUNCE` messages over UDP so that disco-daemon can synchronize the system clock. Three implementations are available: the Go binary (built as part of this project), an Arduino sketch, and an ESPHome component for ESP32/ESP8266.

All implementations broadcast on UDP port 5354, the same port disco uses for host discovery.

## Go binary (Raspberry Pi, Linux)

### Build

```bash
make                                              # builds disco-gps-broadcaster for the current platform
```

Cross-compile for Raspberry Pi Zero 2W (arm64):

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -o build/bin/disco-gps-broadcaster cmd/gps-broadcaster/main.go
```

### Run

```bash
disco-gps-broadcaster -device /dev/ttyACM0
```

With options:

```bash
disco-gps-broadcaster \
  -device /dev/ttyACM0 \
  -broadcast 255.255.255.255:5354 \
  -id gps-pi-01 \
  -interval 16s \
  -interfaces eth0,wlan0 \
  -v
```

| Flag | Description |
|------|-------------|
| `-device` | Serial device path (e.g. `/dev/ttyACM0`) |
| `-broadcast` | Broadcast address (default `255.255.255.255:5354`) |
| `-id` | Source ID included in messages |
| `-interval` | Broadcast interval (default `16s`) |
| `-interfaces` | Comma-separated list of interfaces to broadcast on |
| `-v` | Verbose output |

### systemd service

Create `/etc/systemd/system/disco-gps-broadcaster.service`:

```ini
[Unit]
Description=Disco GPS time broadcaster
After=network.target

[Service]
ExecStart=/usr/local/bin/disco-gps-broadcaster -device /dev/ttyACM0 -id gps-pi-01
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now disco-gps-broadcaster
```

## Arduino

The Arduino sketch is in `gps-broadcaster/arduino/`. It reads NMEA data from a GPS module over a serial connection and broadcasts TIME_ANNOUNCE messages via UDP.

### Hardware

Connect a GPS module to the Arduino's hardware serial pins. The sketch reads from `Serial1` by default.

### Flash

Open `gps-broadcaster/arduino/gps_time_broadcaster.ino` in the Arduino IDE (or use PlatformIO with `gps-broadcaster/arduino/platformio.ini`). Configure your network settings in the sketch before uploading:

```cpp
// Broadcast interval (milliseconds)
const unsigned long BROADCAST_INTERVAL = 16000;
```

## ESPHome (ESP32 / ESP8266)

The ESPHome component is in `gps-broadcaster/esphome/`. Copy both files to your ESPHome configuration directory.

### Hardware (ESP32)

| GPS Module | ESP32 |
|------------|-------|
| VCC | 3.3V |
| GND | GND |
| TX | GPIO16 (RX2) |
| RX | GPIO17 (TX2) |

### Configure and flash

Edit `gps-broadcaster.yaml` to set your Wi-Fi credentials and source ID, then flash:

```bash
esphome run gps-broadcaster.yaml
```

Tune the broadcaster in `gps-broadcaster.yaml`:

```yaml
custom_component:
  - lambda: |-
      auto broadcaster = new GPSBroadcasterComponent();
      broadcaster->set_source_id("gps-esphome-01");
      broadcaster->set_interval(16000);    # ms between broadcasts
      broadcaster->set_port(5354);
      App.register_component(broadcaster);
      return {broadcaster};
```

## Verify

On any node running disco-daemon with `time_sync.enabled: true`, listen for broadcasts:

```bash
sudo tcpdump -i any udp port 5354 -A
```

Then check whether the daemon received a time update:

```bash
disco time
```

`Sources: 1` (or more) confirms the daemon received a TIME_ANNOUNCE message.
