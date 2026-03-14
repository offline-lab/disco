/*
 * GPS Time Broadcaster for nss-daemon
 * 
 * Hardware:
 *   - Arduino Nano/Uno/Mega or ESP32/ESP8266
 *   - GPS module (GNS 701, u-blox NEO-6M, NEO-8M, or similar)
 *   - Ethernet shield (W5100/W5500) or WiFi shield
 * 
 * GNS 701 Pinout:
 *   Pin 1 (VCC)     -> 3.3V (or 5V with level shifter for 5V Arduinos)
 *   Pin 2 (GND)     -> GND
 *   Pin 3 (VBACKUP) -> 3.3V (or battery for RTC backup)
 *   Pin 4 (TX0)     -> Arduino RX (pin 0 or Serial1)
 *   Pin 5 (RX0)     -> Arduino TX (pin 1 or Serial1)
 *   Pin 6 (ENABLE)  -> 3.3V or leave floating
 *   Pin 7 (1PPS)    -> Optional: connect to PPS_PIN for precision timing
 *   Pin 8 (3D_FIX)  -> Optional: LED indicator
 * 
 * NEO-6M/8M Pinout:
 *   VCC  -> 5V or 3.3V (check module specs)
 *   GND  -> GND
 *   TX   -> Arduino RX (pin 0 or Serial1)
 *   RX   -> Arduino TX (pin 1 or Serial1)
 *   PPS  -> Optional: connect to PPS_PIN for precision timing
 * 
 * Build:
 *   Arduino IDE: Open this file and upload
 *   PlatformIO:  pio run -e arduino
 */

#include <SPI.h>

// =============================================================================
// CONFIGURATION - Edit these values for your setup
// =============================================================================

// Network configuration - choose one
#define USE_ETHERNET 1
// #define USE_WIFI 1

// Pin configuration (adjust for your board)
// For Arduino Uno/Nano: GPS uses Serial (pins 0/1), use SoftwareSerial for debug
// For Arduino Mega: GPS can use Serial1/2/3, Serial for debug
// For ESP32: GPS can use any pins via HardwareSerial
#ifndef GPS_SERIAL
#define GPS_SERIAL Serial
#endif

// Optional: 1PPS pin for precision timing (comment out to disable)
// #define PPS_PIN 2

// Network settings
#if USE_ETHERNET
#include <Ethernet.h>
#include <EthernetUDP.h>
byte mac[] = { 0xDE, 0xAD, 0xBE, 0xEF, 0xFE, 0xED };
EthernetUDP udp;
#elif USE_WIFI
#include <WiFi.h>
#include <WiFiUDP.h>
char ssid[] = "YOUR_SSID";
char pass[] = "YOUR_PASSWORD";
WiFiUDP udp;
#endif

// Time broadcast settings
const char SOURCE_ID[] = "gps-arduino-01";
const unsigned int BROADCAST_PORT = 5354;
const unsigned long BROADCAST_INTERVAL = 16000;

// State
unsigned long lastBroadcast = 0;
bool hasFix = false;
unsigned long gpsTimeSeconds = 0;
int satellites = 0;

#ifdef PPS_PIN
// 1PPS state - for precision timing
volatile bool ppsReceived = false;
volatile unsigned long ppsMillis = 0;
volatile unsigned long ppsCount = 0;

void ppsInterrupt() {
  ppsMillis = millis();
  ppsCount++;
  ppsReceived = true;
}
#endif

// NMEA parsing buffer
char nmeaBuffer[128];
int nmeaIndex = 0;

void setup() {
  GPS_SERIAL.begin(9600);  // GPS serial
  
#ifdef PPS_PIN
  pinMode(PPS_PIN, INPUT_PULLUP);
  attachInterrupt(digitalPinToInterrupt(PPS_PIN), ppsInterrupt, RISING);
#endif
  
#if USE_ETHERNET
  Ethernet.begin(mac);
#elif USE_WIFI
  WiFi.begin(ssid, pass);
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
  }
#endif

  udp.begin(BROADCAST_PORT);
}

void loop() {
  // Read GPS data
  while (GPS_SERIAL.available()) {
    char c = GPS_SERIAL.read();
    if (c == '\n') {
      nmeaBuffer[nmeaIndex] = '\0';
      parseNMEA(nmeaBuffer);
      nmeaIndex = 0;
    } else if (c != '\r' && nmeaIndex < sizeof(nmeaBuffer) - 1) {
      nmeaBuffer[nmeaIndex++] = c;
    }
  }

#ifdef PPS_PIN
  // Use 1PPS for more precise broadcast timing when available
  if (ppsReceived && hasFix) {
    ppsReceived = false;
    // Broadcast on PPS pulse (every second) or at interval
    if (millis() - lastBroadcast >= BROADCAST_INTERVAL) {
      broadcastTime();
      lastBroadcast = millis();
    }
  }
#else
  // Broadcast at interval without PPS
  if (hasFix && millis() - lastBroadcast >= BROADCAST_INTERVAL) {
    broadcastTime();
    lastBroadcast = millis();
  }
#endif
}

void parseNMEA(char* line) {
  if (strncmp(line, "$GPRMC,", 7) == 0 || strncmp(line, "$GNRMC,", 7) == 0) {
    parseRMC(line);
  } else if (strncmp(line, "$GPGGA,", 7) == 0 || strncmp(line, "$GNGGA,", 7) == 0) {
    parseGGA(line);
  }
}

void parseRMC(char* line) {
  // $GPRMC,hhmmss.ss,A,ddmm.mmmm,N,dddmm.mmmm,W,x.x,x.x,ddmmyy,x.x,E,A*xx
  char* fields[15];
  int fieldCount = 0;
  
  char* token = strtok(line, ",");
  while (token != NULL && fieldCount < 15) {
    fields[fieldCount++] = token;
    token = strtok(NULL, ",");
  }
  
  if (fieldCount < 10) return;
  
  if (fields[2][0] != 'A') {
    hasFix = false;
    return;
  }
  
  // Parse time (hhmmss.ss) and date (ddmmyy)
  char* timeStr = fields[1];
  char* dateStr = fields[9];
  
  if (strlen(timeStr) < 6 || strlen(dateStr) < 6) return;
  
  int hour = (timeStr[0] - '0') * 10 + (timeStr[1] - '0');
  int minute = (timeStr[2] - '0') * 10 + (timeStr[3] - '0');
  int second = (timeStr[4] - '0') * 10 + (timeStr[5] - '0');
  
  int day = (dateStr[0] - '0') * 10 + (dateStr[1] - '0');
  int month = (dateStr[2] - '0') * 10 + (dateStr[3] - '0');
  int year = 2000 + (dateStr[4] - '0') * 10 + (dateStr[5] - '0');
  
  // Convert to Unix timestamp (simplified, assumes year >= 2000)
  gpsTimeSeconds = toUnixTime(year, month, day, hour, minute, second);
  hasFix = true;
}

void parseGGA(char* line) {
  // $GPGGA,hhmmss.ss,ddmm.mmmm,N,dddmm.mmmm,W,q,ss,y.y,a.a,M,g.g,M,x.x,nnnn*xx
  char* fields[15];
  int fieldCount = 0;
  
  char* token = strtok(line, ",");
  while (token != NULL && fieldCount < 15) {
    fields[fieldCount++] = token;
    token = strtok(NULL, ",");
  }
  
  if (fieldCount < 8) return;
  
  if (fields[6][0] == '0') return;  // No fix
  
  satellites = atoi(fields[7]);
}

unsigned long toUnixTime(int year, int month, int day, int hour, int minute, int second) {
  // Days since Unix epoch (Jan 1, 1970)
  int days = 0;
  
  for (int y = 1970; y < year; y++) {
    days += isLeapYear(y) ? 366 : 365;
  }
  
  int daysInMonth[] = {31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31};
  if (isLeapYear(year)) daysInMonth[1] = 29;
  
  for (int m = 1; m < month; m++) {
    days += daysInMonth[m - 1];
  }
  
  days += day - 1;
  
  return days * 86400UL + hour * 3600UL + minute * 60UL + second;
}

bool isLeapYear(int year) {
  return (year % 4 == 0 && year % 100 != 0) || (year % 400 == 0);
}

void broadcastTime() {
  char json[320];
  unsigned long timestamp = gpsTimeSeconds * 1000000000ULL;  // Convert to nanoseconds
  
#ifdef PPS_PIN
  // Include PPS count for precision timing verification
  snprintf(json, sizeof(json),
    "{\"type\":\"TIME_ANNOUNCE\","
    "\"message_id\":\"%s-%lu\","
    "\"timestamp\":%lu000000000,"
    "\"clock_info\":{\"stratum\":1,\"precision\":-20,\"root_delay\":0.0,\"root_dispersion\":0.0001,\"reference_id\":\"GPS\",\"reference_time\":%lu000000000},"
    "\"leap_indicator\":0,"
    "\"source_id\":\"%s\","
    "\"pps_count\":%lu}",
    SOURCE_ID, millis(), gpsTimeSeconds, gpsTimeSeconds, SOURCE_ID, ppsCount);
#else
  snprintf(json, sizeof(json),
    "{\"type\":\"TIME_ANNOUNCE\","
    "\"message_id\":\"%s-%lu\","
    "\"timestamp\":%lu000000000,"
    "\"clock_info\":{\"stratum\":1,\"precision\":-20,\"root_delay\":0.0,\"root_dispersion\":0.0001,\"reference_id\":\"GPS\",\"reference_time\":%lu000000000},"
    "\"leap_indicator\":0,"
    "\"source_id\":\"%s\"}",
    SOURCE_ID, millis(), gpsTimeSeconds, gpsTimeSeconds, SOURCE_ID);
#endif
  
#if USE_ETHERNET
  udp.beginPacket(Ethernet.broadcastIP(), BROADCAST_PORT);
#else
  udp.beginPacket("255.255.255.255", BROADCAST_PORT);
#endif
  udp.write((uint8_t*)json, strlen(json));
  udp.endPacket();
}
