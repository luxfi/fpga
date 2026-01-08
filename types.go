// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package fpga provides cross-platform FPGA acceleration for Lux Network.
// Supports AMD Versal, AWS F2, Intel Stratix, and Xilinx Alveo FPGAs.
//
// Used by:
//   - DEX: Ultra-low latency order matching
//   - Z-Chain: ZK proof generation (NTT, MSM, Poseidon, FHE)
package fpga

import (
	"time"
	"unsafe"
)

// Backend represents the FPGA backend type
type Backend string

const (
	BackendNone       Backend = "None"
	BackendAMDVersal  Backend = "AMDVersal"
	BackendAWSF2      Backend = "AWSF2"
	BackendIntelStratix Backend = "IntelStratix"
	BackendXilinxAlveo Backend = "XilinxAlveo"
	BackendSimulation Backend = "Simulation"
)

// DeviceType represents specific FPGA device models
type DeviceType int

const (
	DeviceUnknown DeviceType = iota

	// AMD Versal family
	DeviceVersalVE2802  // AI Edge - 400 AI engines
	DeviceVersalVPK180  // Premium - 304 AI engines, 112G
	DeviceVersalVPK280  // Premium - 400 AI engines, 112G
	DeviceVersalVH1782  // HBM series - 400 AI engines, HBM

	// AWS F2 family (Xilinx VU9P based)
	DeviceAWSF2XLarge    // 1 FPGA, 8 vCPUs
	DeviceAWSF22XLarge   // 1 FPGA, 8 vCPUs, 244GB RAM
	DeviceAWSF24XLarge   // 2 FPGAs, 16 vCPUs
	DeviceAWSF216XLarge  // 8 FPGAs, 64 vCPUs

	// Intel Stratix family
	DeviceStratix10GX   // High-performance
	DeviceStratix10MX   // Mid-range with HBM
	DeviceAgilex7       // Latest generation

	// Xilinx Alveo family
	DeviceAlveoU200     // Data center card
	DeviceAlveoU250     // Data center card with HBM
	DeviceAlveoU280     // HBM enabled
)

// Config holds FPGA accelerator configuration
type Config struct {
	// Device selection
	Backend    Backend
	DeviceType DeviceType
	DeviceID   string
	PCIeSlot   string

	// Hardware resources
	AIEngines     int   // AMD Versal AI engines
	DSPSlices     int   // DSP slice count
	DDRChannels   int   // DDR memory channels
	DDRSize       int64 // DDR memory size in bytes
	HBMSize       int64 // HBM memory size in bytes

	// Clock configuration
	KernelClockMHz int // Main kernel clock
	MemoryClockMHz int // Memory clock

	// DMA configuration
	DMAChannels   int
	DMABufferSize int64

	// Network configuration
	Enable100G     bool
	EnableRDMA     bool
	EnableDPDK     bool

	// AWS-specific
	AGFI         string // Amazon FPGA Image ID
	InstanceType string // f2.xlarge, f2.2xlarge, etc.

	// Kernel configuration
	EnableZKKernels  bool // NTT, MSM, Poseidon
	EnableDEXKernels bool // Order matching
	EnableFHEKernels bool // FHE operations
}

// DefaultConfig returns default FPGA configuration
func DefaultConfig() Config {
	return Config{
		Backend:        BackendSimulation,
		KernelClockMHz: 250,
		MemoryClockMHz: 300,
		DMAChannels:    4,
		DMABufferSize:  16 * 1024 * 1024, // 16MB
		EnableZKKernels: true,
	}
}

// Stats holds FPGA performance statistics
type Stats struct {
	// Operations
	OperationsProcessed uint64
	OperationsFailed    uint64

	// Latency (nanoseconds)
	AverageLatencyNs uint64
	MinLatencyNs     uint64
	MaxLatencyNs     uint64
	P99LatencyNs     uint64

	// Throughput
	OpsPerSecond      float64
	ThroughputGbps    float64
	PCIeBandwidthGbps float64
	DMABandwidthGbps  float64

	// Resource utilization
	AIEngineUtil   float64 // AMD Versal
	DSPUtil        float64
	BRAMUtil       float64
	MemoryUsedMB   int64

	// Power and thermal
	PowerWatts       float64
	TemperatureCelsius float64

	// Time
	LastUpdate time.Time
	Uptime     time.Duration
}

// DMADirection for FPGA transfers
type DMADirection int

const (
	DMAToDevice DMADirection = iota
	DMAFromDevice
	DMABidirectional
)

// DMADescriptor for scatter-gather DMA
type DMADescriptor struct {
	SrcAddr  uint64
	DstAddr  uint64
	Length   uint32
	Control  uint32
	NextDesc uint64
	Status   uint32
}

// DMABuffer represents a DMA-capable memory buffer
type DMABuffer struct {
	VirtualAddr  unsafe.Pointer
	PhysicalAddr uint64
	Size         int64
	Direction    DMADirection
	IsMapped     bool
}

// KernelType represents FPGA kernel types
type KernelType int

const (
	KernelNone KernelType = iota

	// ZK kernels (Z-Chain)
	KernelNTT          // Number Theoretic Transform
	KernelINTT         // Inverse NTT
	KernelMSM          // Multi-Scalar Multiplication
	KernelPoseidon     // Poseidon hash
	KernelPoseidon2    // Poseidon2 hash
	KernelFHEAdd       // FHE addition
	KernelFHEMul       // FHE multiplication
	KernelFHEBootstrap // FHE bootstrapping

	// DEX kernels
	KernelOrderMatch   // Order matching engine
	KernelRiskCheck    // Risk checking
	KernelOrderBook    // Order book management

	// Common kernels
	KernelDMA          // DMA transfer
	KernelMemcpy       // Memory copy
)

// Kernel represents an FPGA compute kernel
type Kernel struct {
	Type       KernelType
	Name       string
	BaseAddr   uint64
	Size       int64
	IsLoaded   bool
	IsRunning  bool
	ClockMHz   int
	InputSize  int
	OutputSize int
}

// Accelerator is the main interface for FPGA acceleration
type Accelerator interface {
	// Lifecycle
	Initialize(config Config) error
	Shutdown() error
	Reset() error

	// Health and monitoring
	IsHealthy() bool
	GetStats() *Stats
	GetTemperature() float64
	GetPowerUsage() float64

	// Device info
	Backend() Backend
	DeviceType() DeviceType
	DeviceID() string
	Capabilities() Capabilities

	// Kernel management
	LoadKernel(kernelType KernelType, bitstream []byte) error
	UnloadKernel(kernelType KernelType) error
	IsKernelLoaded(kernelType KernelType) bool

	// DMA operations
	AllocateDMABuffer(size int64, direction DMADirection) (*DMABuffer, error)
	FreeDMABuffer(buffer *DMABuffer) error
	DMATransfer(src, dst *DMABuffer, size int64) error
	DMATransferAsync(src, dst *DMABuffer, size int64) (<-chan error, error)

	// Generic execution
	Execute(kernel KernelType, input, output []byte) error
	ExecuteAsync(kernel KernelType, input []byte) (<-chan []byte, error)

	// Clock management
	SetKernelClock(mhz int) error
	GetKernelClock() int
}

// Capabilities describes what the FPGA supports
type Capabilities struct {
	// Hardware
	MaxKernelClockMHz int
	MaxMemoryClockMHz int
	MaxDMAChannels    int
	MaxDMABufferSize  int64

	// Resources
	LogicCells    int
	DSPSlices     int
	BRAMBlocks    int
	URAMBlocks    int
	AIEngines     int // AMD Versal
	DDRChannels   int
	DDRSizeGB     int
	HBMSizeGB     int

	// PCIe
	PCIeGen    int
	PCIeLanes  int
	PCIeGbps   float64

	// Network
	Has100GEthernet bool
	HasRDMA         bool
	Has400GEthernet bool

	// Supported kernels
	SupportedKernels []KernelType
}

// ZKAccelerator extends Accelerator with ZK-specific operations
type ZKAccelerator interface {
	Accelerator

	// Number Theoretic Transform
	NTT(input []uint64, logN int, forward bool) ([]uint64, error)
	NTTBatch(inputs [][]uint64, logN int, forward bool) ([][]uint64, error)

	// Multi-Scalar Multiplication
	MSM(pointsX, pointsY, scalars []uint64, count int) ([]uint64, error)
	MSMBatch(pointsX, pointsY, scalars [][]uint64) ([][]uint64, error)

	// Poseidon hashing
	PoseidonHash(input []uint64, rate int) ([]uint64, error)
	PoseidonHashBatch(inputs [][]uint64, rate int) ([][]uint64, error)

	// FHE operations
	FHEAdd(ctA, ctB []byte) ([]byte, error)
	FHEMul(ctA, ctB []byte) ([]byte, error)
	FHEBootstrap(ct []byte) ([]byte, error)
}

// DEXAccelerator extends Accelerator with DEX-specific operations
type DEXAccelerator interface {
	Accelerator

	// Order processing
	ProcessOrder(order *Order) (*OrderResult, error)
	ProcessOrderBatch(orders []*Order) ([]*OrderResult, error)
	CancelOrder(orderID uint64) error

	// Order book
	UpdateOrderBook(symbol string, bids, asks []PriceLevel) error
	GetOrderBook(symbol string) (*OrderBookSnapshot, error)
}

// Order represents a trading order (DEX)
type Order struct {
	OrderID   uint64
	Symbol    uint32
	Side      uint8 // 0=buy, 1=sell
	Type      uint8 // 0=limit, 1=market
	Price     uint64
	Quantity  uint64
	Timestamp uint64
	UserID    uint32
	Flags     uint32
}

// OrderResult represents order processing result
type OrderResult struct {
	OrderID       uint64
	Status        uint8 // 0=accepted, 1=rejected, 2=filled, 3=partial
	ExecutedQty   uint64
	ExecutedPrice uint64
	TradeID       uint64
	MatchLatency  uint32
	Timestamp     uint64
}

// PriceLevel for order book
type PriceLevel struct {
	Price    uint64
	Quantity uint64
	Orders   uint32
}

// OrderBookSnapshot represents a point-in-time order book state
type OrderBookSnapshot struct {
	Symbol    string
	Bids      []PriceLevel
	Asks      []PriceLevel
	Timestamp uint64
	Sequence  uint64
}
