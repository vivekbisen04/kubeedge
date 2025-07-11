package util

import (
    "fmt"
    "os"
)

// FormatMessage formats a message with prefix
func FormatMessage(prefix, message string) string {
    if prefix == "" {
        return message
    }
    return fmt.Sprintf("[%s] %s", prefix, message)
}

// ReadConfigFile reads configuration from file
func ReadConfigFile(filename string) ([]byte, error) {
    if filename == "" {
        return nil, fmt.Errorf("filename cannot be empty")
    }
    return os.ReadFile(filename)
}

// ValidatePort validates if port is in valid range
func ValidatePort(port int) error {
    if port < 1 || port > 65535 {
        return fmt.Errorf("port %d is out of valid range (1-65535)", port)
    }
    return nil
}
