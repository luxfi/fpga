// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package fpga

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrNoFPGASupport     = errors.New("FPGA support not compiled (use -tags fpga)")
	ErrDeviceNotFound    = errors.New("FPGA device not found")
	ErrInvalidBackend    = errors.New("invalid FPGA backend")
	ErrNotInitialized    = errors.New("FPGA accelerator not initialized")
	ErrKernelNotLoaded   = errors.New("FPGA kernel not loaded")
	ErrDMAFailed         = errors.New("DMA transfer failed")
	ErrUnsupportedDevice = errors.New("unsupported FPGA device")
)

// registry holds registered FPGA backends
var (
	registry = make(map[Backend]func() Accelerator)
	regMu    sync.RWMutex
)

// RegisterBackend registers an FPGA backend factory
func RegisterBackend(backend Backend, factory func() Accelerator) {
	regMu.Lock()
	defer regMu.Unlock()
	registry[backend] = factory
}

// GetBackends returns all registered backends
func GetBackends() []Backend {
	regMu.RLock()
	defer regMu.RUnlock()

	backends := make([]Backend, 0, len(registry))
	for b := range registry {
		backends = append(backends, b)
	}
	return backends
}

// NewAccelerator creates an FPGA accelerator based on configuration
func NewAccelerator(config Config) (Accelerator, error) {
	regMu.RLock()
	factory, ok := registry[config.Backend]
	regMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidBackend, config.Backend)
	}

	acc := factory()
	if err := acc.Initialize(config); err != nil {
		return nil, fmt.Errorf("failed to initialize %s: %w", config.Backend, err)
	}

	return acc, nil
}

// NewZKAccelerator creates a ZK-specific FPGA accelerator
func NewZKAccelerator(config Config) (ZKAccelerator, error) {
	config.EnableZKKernels = true

	acc, err := NewAccelerator(config)
	if err != nil {
		return nil, err
	}

	zkAcc, ok := acc.(ZKAccelerator)
	if !ok {
		acc.Shutdown()
		return nil, fmt.Errorf("backend %s does not support ZK operations", config.Backend)
	}

	return zkAcc, nil
}

// NewDEXAccelerator creates a DEX-specific FPGA accelerator
func NewDEXAccelerator(config Config) (DEXAccelerator, error) {
	config.EnableDEXKernels = true

	acc, err := NewAccelerator(config)
	if err != nil {
		return nil, err
	}

	dexAcc, ok := acc.(DEXAccelerator)
	if !ok {
		acc.Shutdown()
		return nil, fmt.Errorf("backend %s does not support DEX operations", config.Backend)
	}

	return dexAcc, nil
}

// DetectDevices scans for available FPGA devices
func DetectDevices() ([]DeviceInfo, error) {
	var devices []DeviceInfo

	// Scan PCIe bus for known FPGA vendors
	// AMD/Xilinx: 0x10ee
	// Intel/Altera: 0x1172
	// AWS F2: Custom shell

	// This would interface with /sys/bus/pci/devices on Linux
	// For now, return empty list (simulation mode)

	return devices, nil
}

// DeviceInfo holds detected FPGA device information
type DeviceInfo struct {
	Backend     Backend
	DeviceType  DeviceType
	DeviceID    string
	PCIeSlot    string
	VendorID    uint16
	DeviceIDHex uint16
	DriverName  string
	IsAvailable bool
}

// AutoDetectBackend automatically selects the best available backend
func AutoDetectBackend() (Backend, error) {
	devices, err := DetectDevices()
	if err != nil {
		return BackendNone, err
	}

	// Priority: AMD Versal > AWS F2 > Intel Stratix > Xilinx Alveo
	for _, dev := range devices {
		if dev.IsAvailable && dev.Backend == BackendAMDVersal {
			return BackendAMDVersal, nil
		}
	}

	for _, dev := range devices {
		if dev.IsAvailable && dev.Backend == BackendAWSF2 {
			return BackendAWSF2, nil
		}
	}

	for _, dev := range devices {
		if dev.IsAvailable && dev.Backend == BackendIntelStratix {
			return BackendIntelStratix, nil
		}
	}

	for _, dev := range devices {
		if dev.IsAvailable && dev.Backend == BackendXilinxAlveo {
			return BackendXilinxAlveo, nil
		}
	}

	// Fall back to simulation
	return BackendSimulation, nil
}
