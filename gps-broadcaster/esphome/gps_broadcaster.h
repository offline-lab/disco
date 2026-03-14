// GPS Broadcaster Component for ESPHome
// Place this file in your ESPHome config directory alongside gps-broadcaster.yaml

#pragma once

#include "esphome.h"
#include <WiFiUdp.h>

static const char *const TAG = "gps_broadcaster";

class GPSBroadcasterComponent : public Component {
public:
    GPSBroadcasterComponent() {}
    
    void setup() override {
        udp_.stop();
        last_broadcast_ = 0;
        ESP_LOGI(TAG, "GPS Broadcaster initialized (source: %s, interval: %lums, port: %u)",
                 source_id_.c_str(), interval_ms_, port_);
    }
    
    void loop() override {
        uint32_t now = millis();
        
        // Get GPS time from ESPHome's time component
        auto time = id(gps_time).now();
        if (!time.is_valid()) {
            return;
        }
        
        // Broadcast at interval
        if (now - last_broadcast_ >= interval_ms_ || last_broadcast_ == 0) {
            broadcast_time(time);
            last_broadcast_ = now;
        }
    }
    
    void broadcast_time(const ESPTime &time) {
        if (!time.is_valid()) {
            ESP_LOGD(TAG, "No valid GPS time");
            return;
        }
        
        // Calculate Unix timestamp in nanoseconds
        uint64_t timestamp_ns = (uint64_t)time.timestamp * 1000000000ULL;
        
        // Build JSON message
        char json[512];
        snprintf(json, sizeof(json),
            "{"
            "\"type\":\"TIME_ANNOUNCE\","
            "\"message_id\":\"%s-%lld\","
            "\"timestamp\":%lld,"
            "\"clock_info\":{"
            "\"stratum\":1,"
            "\"precision\":-20,"
            "\"root_delay\":0.0,"
            "\"root_dispersion\":0.0001,"
            "\"reference_id\":\"GPS\","
            "\"reference_time\":%lld"
            "},"
            "\"leap_indicator\":0,"
            "\"source_id\":\"%s\""
            "}",
            source_id_.c_str(),
            (long long)time.timestamp,
            (long long)timestamp_ns,
            (long long)timestamp_ns,
            source_id_.c_str()
        );
        
        // Broadcast to network
        udp_.stop();
        if (!udp_.beginPacket(IPADDR_BROADCAST, port_)) {
            ESP_LOGE(TAG, "Failed to begin UDP packet");
            return;
        }
        
        udp_.write((const uint8_t*)json, strlen(json));
        int result = udp_.endPacket();
        
        if (result == 0) {
            ESP_LOGE(TAG, "Failed to send UDP packet");
            return;
        }
        
        ESP_LOGI(TAG, "Broadcast GPS time: %04d-%02d-%02d %02d:%02d:%02d UTC",
            time.year, time.month, time.day_of_month,
            time.hour, time.minute, time.second);
    }
    
    void set_source_id(const std::string &id) { source_id_ = id; }
    void set_interval(uint32_t ms) { interval_ms_ = ms; }
    void set_port(uint16_t port) { port_ = port; }
    
    float get_setup_priority() const override { 
        return esphome::setup_priority::AFTER_CONNECTION; 
    }
    
private:
    WiFiUDP udp_;
    std::string source_id_ = "gps-esphome-01";
    uint32_t interval_ms_ = 16000;
    uint16_t port_ = 5354;
    uint32_t last_broadcast_ = 0;
};
