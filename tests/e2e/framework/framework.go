// Package framework provides test utilities for E2E scenario execution.
// It handles gateway and simulator API interaction, event waiting, and assertion helpers.
package framework

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config holds E2E test framework configuration.
type Config struct {
	GatewayURL      string
	CameraSimURL    string
	SensorSimURL    string
	EquipmentSimURL string
	OutputSimURL    string
	ModbusSimURL    string
	ScenarioURL     string
	Timeout         time.Duration
}

// DefaultConfig returns configuration for local testing.
func DefaultConfig() Config {
	return Config{
		GatewayURL:      "http://localhost:8080",
		CameraSimURL:    "http://localhost:9001",
		SensorSimURL:    "http://localhost:9002",
		EquipmentSimURL: "http://localhost:9003",
		OutputSimURL:    "http://localhost:9004",
		ModbusSimURL:    "http://localhost:9005",
		ScenarioURL:     "http://localhost:9010",
		Timeout:         30 * time.Second,
	}
}

// Client provides methods to interact with the gateway and simulators.
type Client struct {
	config Config
	http   *http.Client
}

// NewClient creates a new E2E test client.
func NewClient(config Config) *Client {
	return &Client{
		config: config,
		http: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// HealthCheck verifies that a service is healthy.
func (c *Client) HealthCheck(url string) error {
	resp, err := c.http.Get(url + "/health")
	if err != nil {
		return fmt.Errorf("health check failed for %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned %d for %s", resp.StatusCode, url)
	}
	return nil
}

// WaitForService waits until a service responds to health checks.
func (c *Client) WaitForService(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := c.HealthCheck(url); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("service %s did not become healthy within %s", url, timeout)
}

// SetModbusDI sets a digital input on the Modbus simulator.
func (c *Client) SetModbusDI(index int, value bool) error {
	body := fmt.Sprintf(`{"index":%d,"value":%v}`, index, value)
	resp, err := c.http.Post(
		c.config.ModbusSimURL+"/di",
		"application/json",
		strings.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("set modbus DI failed: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// ExecuteOutput sends an output command to the output simulator.
func (c *Client) ExecuteOutput(commandType, target string) (map[string]interface{}, error) {
	body := fmt.Sprintf(`{"commandId":"test-%d","commandType":"%s","target":"%s","createdAt":"%s"}`,
		time.Now().UnixMilli(), commandType, target, time.Now().UTC().Format(time.RFC3339))

	resp, err := c.http.Post(
		c.config.OutputSimURL+"/execute",
		"application/json",
		strings.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("execute output failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode output result failed: %w", err)
	}
	return result, nil
}

// GetGatewayHealth returns the gateway health status.
func (c *Client) GetGatewayHealth() (map[string]interface{}, error) {
	return c.getJSON(c.config.GatewayURL + "/health/ready")
}

// GetCameraEvents returns recent camera events.
func (c *Client) GetCameraEvents() ([]map[string]interface{}, error) {
	return c.getJSONArray(c.config.CameraSimURL + "/events")
}

// GetSensorReadings returns recent sensor readings.
func (c *Client) GetSensorReadings() ([]map[string]interface{}, error) {
	return c.getJSONArray(c.config.SensorSimURL + "/readings")
}

// GetEquipmentStatus returns current equipment states.
func (c *Client) GetEquipmentStatus() ([]map[string]interface{}, error) {
	return c.getJSONArray(c.config.EquipmentSimURL + "/status")
}

// GetModbusRegisters returns current Modbus DI/DO states.
func (c *Client) GetModbusRegisters() (map[string]interface{}, error) {
	return c.getJSON(c.config.ModbusSimURL + "/registers")
}

func (c *Client) getJSON(url string) (map[string]interface{}, error) {
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) getJSONArray(url string) ([]map[string]interface{}, error) {
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
