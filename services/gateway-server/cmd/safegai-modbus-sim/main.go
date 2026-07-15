// Package main implements a Modbus TCP simulator for SafeGAI.
// It provides 8 digital inputs (DI) and 8 digital outputs (DO) over
// a simplified Modbus TCP protocol for integration testing.
// The simulator listens on TCP and provides an HTTP API for inspection.
package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	defaultModbusAddr = ":5020"
	defaultHTTPAddr   = ":9005"
	numDI             = 8
	numDO             = 8
	version           = "0.1.0"

	// Modbus function codes
	fcReadCoils          = 0x01
	fcReadDiscreteInputs = 0x02
	fcWriteSingleCoil    = 0x05
	fcWriteMultiCoils    = 0x0F
)

// SimState tracks the Modbus register state.
type SimState struct {
	mu             sync.RWMutex
	running        bool
	digitalInputs  [numDI]bool
	digitalOutputs [numDO]bool
	totalReads     int64
	totalWrites    int64
	startTime      time.Time
}

var state = &SimState{}

func main() {
	modbusAddr := envOrDefault("MODBUS_SIM_ADDR", defaultModbusAddr)
	httpAddr := envOrDefault("MODBUS_SIM_HTTP_ADDR", defaultHTTPAddr)

	logJSON("info", "SafeGAI Modbus TCP simulator starting", map[string]string{
		"modbusAddr": modbusAddr,
		"httpAddr":   httpAddr,
		"DI":         fmt.Sprintf("%d", numDI),
		"DO":         fmt.Sprintf("%d", numDO),
	})

	state.startTime = time.Now()
	state.running = true

	// Set some initial DI states
	state.digitalInputs[0] = true // Safety door closed
	state.digitalInputs[1] = true // Emergency stop NOT pressed (normally closed)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start Modbus TCP listener
	go startModbusTCP(ctx, modbusAddr)

	// Start HTTP inspection API
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/registers", handleRegisters)
	mux.HandleFunc("/di", handleSetDI)

	server := &http.Server{
		Addr:         httpAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logJSON("error", fmt.Sprintf("HTTP server error: %v", err), nil)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logJSON("info", "Shutting down Modbus simulator", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	state.mu.Lock()
	state.running = false
	state.mu.Unlock()

	logJSON("info", "Modbus simulator stopped", nil)
}

func startModbusTCP(ctx context.Context, addr string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logJSON("error", fmt.Sprintf("Failed to listen on %s: %v", addr, err), nil)
		os.Exit(1)
	}
	defer listener.Close()

	logJSON("info", fmt.Sprintf("Modbus TCP listening on %s", addr), nil)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				logJSON("error", fmt.Sprintf("Accept error: %v", err), nil)
				continue
			}
		}
		go handleModbusConnection(ctx, conn)
	}
}

func handleModbusConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 256)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				logJSON("debug", fmt.Sprintf("Connection read error: %v", err), nil)
			}
			return
		}

		if n < 8 {
			continue // Invalid MBAP header
		}

		// Parse MBAP header (7 bytes) + PDU
		transactionID := binary.BigEndian.Uint16(buf[0:2])
		// protocolID := binary.BigEndian.Uint16(buf[2:4]) // Should be 0
		// length := binary.BigEndian.Uint16(buf[4:6])
		// unitID := buf[6]
		functionCode := buf[7]

		response := processModbusRequest(functionCode, buf[8:n])

		// Build response with MBAP header
		resp := make([]byte, 7+len(response))
		binary.BigEndian.PutUint16(resp[0:2], transactionID)
		binary.BigEndian.PutUint16(resp[2:4], 0) // Protocol ID
		binary.BigEndian.PutUint16(resp[4:6], uint16(1+len(response)))
		resp[6] = 1 // Unit ID
		copy(resp[7:], response)

		conn.Write(resp)
	}
}

func processModbusRequest(fc byte, data []byte) []byte {
	switch fc {
	case fcReadDiscreteInputs:
		if len(data) < 4 {
			return []byte{fc | 0x80, 0x03} // Illegal data value
		}
		startAddr := binary.BigEndian.Uint16(data[0:2])
		quantity := binary.BigEndian.Uint16(data[2:4])

		state.mu.RLock()
		defer state.mu.RUnlock()
		state.totalReads++

		byteCount := (quantity + 7) / 8
		result := make([]byte, 2+byteCount)
		result[0] = fc
		result[1] = byte(byteCount)

		for i := uint16(0); i < quantity; i++ {
			idx := startAddr + i
			if idx < numDI && state.digitalInputs[idx] {
				result[2+i/8] |= 1 << (i % 8)
			}
		}
		return result

	case fcReadCoils:
		if len(data) < 4 {
			return []byte{fc | 0x80, 0x03}
		}
		startAddr := binary.BigEndian.Uint16(data[0:2])
		quantity := binary.BigEndian.Uint16(data[2:4])

		state.mu.RLock()
		defer state.mu.RUnlock()
		state.totalReads++

		byteCount := (quantity + 7) / 8
		result := make([]byte, 2+byteCount)
		result[0] = fc
		result[1] = byte(byteCount)

		for i := uint16(0); i < quantity; i++ {
			idx := startAddr + i
			if idx < numDO && state.digitalOutputs[idx] {
				result[2+i/8] |= 1 << (i % 8)
			}
		}
		return result

	case fcWriteSingleCoil:
		if len(data) < 4 {
			return []byte{fc | 0x80, 0x03}
		}
		addr := binary.BigEndian.Uint16(data[0:2])
		value := binary.BigEndian.Uint16(data[2:4])

		state.mu.Lock()
		defer state.mu.Unlock()
		state.totalWrites++

		if addr < numDO {
			state.digitalOutputs[addr] = (value == 0xFF00)
		}

		// Echo request as response
		result := make([]byte, 5)
		result[0] = fc
		copy(result[1:], data[0:4])
		return result

	case fcWriteMultiCoils:
		if len(data) < 5 {
			return []byte{fc | 0x80, 0x03}
		}
		startAddr := binary.BigEndian.Uint16(data[0:2])
		quantity := binary.BigEndian.Uint16(data[2:4])

		state.mu.Lock()
		defer state.mu.Unlock()
		state.totalWrites++

		for i := uint16(0); i < quantity; i++ {
			idx := startAddr + i
			if idx < numDO && int(5+i/8) < len(data) {
				bit := data[5+i/8] & (1 << (i % 8))
				state.digitalOutputs[idx] = (bit != 0)
			}
		}

		result := make([]byte, 5)
		result[0] = fc
		copy(result[1:5], data[0:4])
		return result

	default:
		return []byte{fc | 0x80, 0x01} // Illegal function
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	running := state.running
	state.mu.RUnlock()

	status := "healthy"
	if !running {
		status = "stopping"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"version": version,
		"uptime":  time.Since(state.startTime).Truncate(time.Second).String(),
		"DI":      numDI,
		"DO":      numDO,
	})
}

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	reads := state.totalReads
	writes := state.totalWrites
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalReads":  reads,
		"totalWrites": writes,
		"uptime":      time.Since(state.startTime).Truncate(time.Second).String(),
	})
}

func handleRegisters(w http.ResponseWriter, _ *http.Request) {
	state.mu.RLock()
	di := state.digitalInputs
	do := state.digitalOutputs
	state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"digitalInputs":  di,
		"digitalOutputs": do,
	})
}

func handleSetDI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Index int  `json:"index"`
		Value bool `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Index < 0 || req.Index >= numDI {
		http.Error(w, "Index out of range", http.StatusBadRequest)
		return
	}

	state.mu.Lock()
	state.digitalInputs[req.Index] = req.Value
	state.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"index": req.Index,
		"value": req.Value,
	})
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func logJSON(level, message string, fields map[string]string) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   message,
		"component": "modbus-sim",
	}
	for k, v := range fields {
		entry[k] = v
	}
	data, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stderr, string(data))
}
