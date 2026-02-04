// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package fpga

import (
	"sync"
	"time"
)

func init() {
	RegisterBackend(BackendSimulation, func() Accelerator {
		return NewSimulationAccelerator()
	})
}

// SimulationAccelerator provides a software simulation of FPGA operations
// Used for development, testing, and systems without FPGA hardware
type SimulationAccelerator struct {
	config  Config
	stats   Stats
	kernels map[KernelType]*Kernel
	buffers []*DMABuffer
	mu      sync.RWMutex

	initialized bool
	startTime   time.Time
}

// NewSimulationAccelerator creates a new simulation accelerator
func NewSimulationAccelerator() *SimulationAccelerator {
	return &SimulationAccelerator{
		kernels: make(map[KernelType]*Kernel),
		buffers: make([]*DMABuffer, 0),
	}
}

func (s *SimulationAccelerator) Initialize(config Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config
	s.initialized = true
	s.startTime = time.Now()

	return nil
}

func (s *SimulationAccelerator) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initialized = false
	s.kernels = make(map[KernelType]*Kernel)
	s.buffers = nil

	return nil
}

func (s *SimulationAccelerator) Reset() error {
	if err := s.Shutdown(); err != nil {
		return err
	}
	return s.Initialize(s.config)
}

func (s *SimulationAccelerator) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

func (s *SimulationAccelerator) GetStats() *Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := s.stats
	stats.LastUpdate = time.Now()
	stats.Uptime = time.Since(s.startTime)

	return &stats
}

func (s *SimulationAccelerator) GetTemperature() float64 {
	return 45.0 // Simulated temperature
}

func (s *SimulationAccelerator) GetPowerUsage() float64 {
	return 75.0 // Simulated power in watts
}

func (s *SimulationAccelerator) Backend() Backend {
	return BackendSimulation
}

func (s *SimulationAccelerator) DeviceType() DeviceType {
	return DeviceUnknown
}

func (s *SimulationAccelerator) DeviceID() string {
	return "simulation-0"
}

func (s *SimulationAccelerator) Capabilities() Capabilities {
	return Capabilities{
		MaxKernelClockMHz: 500,
		MaxMemoryClockMHz: 300,
		MaxDMAChannels:    4,
		MaxDMABufferSize:  256 * 1024 * 1024, // 256MB
		LogicCells:        2500000,
		DSPSlices:         6840,
		BRAMBlocks:        4320,
		PCIeGen:           4,
		PCIeLanes:         16,
		PCIeGbps:          32.0,
		SupportedKernels: []KernelType{
			KernelNTT, KernelINTT, KernelMSM, KernelPoseidon, KernelPoseidon2,
			KernelFHEAdd, KernelFHEMul, KernelFHEBootstrap,
			KernelOrderMatch, KernelRiskCheck, KernelOrderBook,
		},
	}
}

func (s *SimulationAccelerator) LoadKernel(kernelType KernelType, bitstream []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.kernels[kernelType] = &Kernel{
		Type:     kernelType,
		Name:     kernelTypeName(kernelType),
		IsLoaded: true,
		ClockMHz: s.config.KernelClockMHz,
	}

	return nil
}

func (s *SimulationAccelerator) UnloadKernel(kernelType KernelType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.kernels, kernelType)
	return nil
}

func (s *SimulationAccelerator) IsKernelLoaded(kernelType KernelType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	kernel, ok := s.kernels[kernelType]
	return ok && kernel.IsLoaded
}

func (s *SimulationAccelerator) AllocateDMABuffer(size int64, direction DMADirection) (*DMABuffer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := make([]byte, size)
	buffer := &DMABuffer{
		VirtualAddr:  &data[0],
		PhysicalAddr: uint64(len(s.buffers)),
		Size:         size,
		Direction:    direction,
		IsMapped:     true,
	}

	s.buffers = append(s.buffers, buffer)
	return buffer, nil
}

func (s *SimulationAccelerator) FreeDMABuffer(buffer *DMABuffer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	buffer.IsMapped = false
	return nil
}

func (s *SimulationAccelerator) DMATransfer(src, dst *DMABuffer, size int64) error {
	// Simulate DMA transfer latency
	time.Sleep(time.Microsecond)

	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.mu.Unlock()

	return nil
}

func (s *SimulationAccelerator) DMATransferAsync(src, dst *DMABuffer, size int64) (<-chan error, error) {
	ch := make(chan error, 1)

	go func() {
		err := s.DMATransfer(src, dst, size)
		ch <- err
		close(ch)
	}()

	return ch, nil
}

func (s *SimulationAccelerator) Execute(kernel KernelType, input, output []byte) error {
	if !s.IsKernelLoaded(kernel) {
		return ErrKernelNotLoaded
	}

	start := time.Now()

	// Simulate kernel execution time based on type
	switch kernel {
	case KernelNTT, KernelINTT:
		time.Sleep(100 * time.Microsecond)
	case KernelMSM:
		time.Sleep(500 * time.Microsecond)
	case KernelPoseidon, KernelPoseidon2:
		time.Sleep(50 * time.Microsecond)
	case KernelFHEAdd:
		time.Sleep(10 * time.Microsecond)
	case KernelFHEMul:
		time.Sleep(100 * time.Microsecond)
	case KernelFHEBootstrap:
		time.Sleep(2 * time.Millisecond)
	case KernelOrderMatch:
		time.Sleep(1 * time.Microsecond)
	default:
		time.Sleep(10 * time.Microsecond)
	}

	latency := time.Since(start)

	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.stats.AverageLatencyNs = uint64(latency.Nanoseconds())
	s.mu.Unlock()

	// Copy input to output for simulation
	copy(output, input)

	return nil
}

func (s *SimulationAccelerator) ExecuteAsync(kernel KernelType, input []byte) (<-chan []byte, error) {
	if !s.IsKernelLoaded(kernel) {
		return nil, ErrKernelNotLoaded
	}

	ch := make(chan []byte, 1)

	go func() {
		output := make([]byte, len(input))
		_ = s.Execute(kernel, input, output)
		ch <- output
		close(ch)
	}()

	return ch, nil
}

func (s *SimulationAccelerator) SetKernelClock(mhz int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config.KernelClockMHz = mhz
	return nil
}

func (s *SimulationAccelerator) GetKernelClock() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config.KernelClockMHz
}

// ZK operations (implements ZKAccelerator)

func (s *SimulationAccelerator) NTT(input []uint64, logN int, forward bool) ([]uint64, error) {
	// Simulated NTT - in production would use FPGA
	output := make([]uint64, len(input))
	copy(output, input)

	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.mu.Unlock()

	return output, nil
}

func (s *SimulationAccelerator) NTTBatch(inputs [][]uint64, logN int, forward bool) ([][]uint64, error) {
	outputs := make([][]uint64, len(inputs))
	for i, input := range inputs {
		out, err := s.NTT(input, logN, forward)
		if err != nil {
			return nil, err
		}
		outputs[i] = out
	}
	return outputs, nil
}

func (s *SimulationAccelerator) MSM(pointsX, pointsY, scalars []uint64, count int) ([]uint64, error) {
	// Simulated MSM - returns placeholder result
	result := make([]uint64, 8) // X and Y coordinates (4 limbs each)

	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.mu.Unlock()

	return result, nil
}

func (s *SimulationAccelerator) MSMBatch(pointsX, pointsY, scalars [][]uint64) ([][]uint64, error) {
	outputs := make([][]uint64, len(pointsX))
	for i := range pointsX {
		out, err := s.MSM(pointsX[i], pointsY[i], scalars[i], len(pointsX[i])/4)
		if err != nil {
			return nil, err
		}
		outputs[i] = out
	}
	return outputs, nil
}

func (s *SimulationAccelerator) PoseidonHash(input []uint64, rate int) ([]uint64, error) {
	// Simulated Poseidon hash
	output := make([]uint64, 4) // 256-bit hash

	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.mu.Unlock()

	return output, nil
}

func (s *SimulationAccelerator) PoseidonHashBatch(inputs [][]uint64, rate int) ([][]uint64, error) {
	outputs := make([][]uint64, len(inputs))
	for i, input := range inputs {
		out, err := s.PoseidonHash(input, rate)
		if err != nil {
			return nil, err
		}
		outputs[i] = out
	}
	return outputs, nil
}

func (s *SimulationAccelerator) FHEAdd(ctA, ctB []byte) ([]byte, error) {
	// Simulated FHE add
	output := make([]byte, len(ctA))
	copy(output, ctA)

	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.mu.Unlock()

	return output, nil
}

func (s *SimulationAccelerator) FHEMul(ctA, ctB []byte) ([]byte, error) {
	// Simulated FHE mul
	output := make([]byte, len(ctA))
	copy(output, ctA)

	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.mu.Unlock()

	return output, nil
}

func (s *SimulationAccelerator) FHEBootstrap(ct []byte) ([]byte, error) {
	// Simulated FHE bootstrap
	output := make([]byte, len(ct))
	copy(output, ct)

	// Bootstrapping is slow even in simulation
	time.Sleep(time.Millisecond)

	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.mu.Unlock()

	return output, nil
}

// DEX operations (implements DEXAccelerator)

func (s *SimulationAccelerator) ProcessOrder(order *Order) (*OrderResult, error) {
	result := &OrderResult{
		OrderID:       order.OrderID,
		Status:        2, // Filled
		ExecutedQty:   order.Quantity,
		ExecutedPrice: order.Price,
		Timestamp:     uint64(time.Now().UnixNano()),
	}

	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.mu.Unlock()

	return result, nil
}

func (s *SimulationAccelerator) ProcessOrderBatch(orders []*Order) ([]*OrderResult, error) {
	results := make([]*OrderResult, len(orders))
	for i, order := range orders {
		result, err := s.ProcessOrder(order)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (s *SimulationAccelerator) CancelOrder(orderID uint64) error {
	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.mu.Unlock()
	return nil
}

func (s *SimulationAccelerator) UpdateOrderBook(symbol string, bids, asks []PriceLevel) error {
	s.mu.Lock()
	s.stats.OperationsProcessed++
	s.mu.Unlock()
	return nil
}

func (s *SimulationAccelerator) GetOrderBook(symbol string) (*OrderBookSnapshot, error) {
	return &OrderBookSnapshot{
		Symbol:    symbol,
		Bids:      []PriceLevel{},
		Asks:      []PriceLevel{},
		Timestamp: uint64(time.Now().UnixNano()),
	}, nil
}

// Helper function
func kernelTypeName(kt KernelType) string {
	names := map[KernelType]string{
		KernelNTT:          "ntt",
		KernelINTT:         "intt",
		KernelMSM:          "msm",
		KernelPoseidon:     "poseidon",
		KernelPoseidon2:    "poseidon2",
		KernelFHEAdd:       "fhe_add",
		KernelFHEMul:       "fhe_mul",
		KernelFHEBootstrap: "fhe_bootstrap",
		KernelOrderMatch:   "order_match",
		KernelRiskCheck:    "risk_check",
		KernelOrderBook:    "order_book",
	}

	if name, ok := names[kt]; ok {
		return name
	}
	return "unknown"
}

// Verify SimulationAccelerator implements all interfaces
var _ Accelerator = (*SimulationAccelerator)(nil)
var _ ZKAccelerator = (*SimulationAccelerator)(nil)
var _ DEXAccelerator = (*SimulationAccelerator)(nil)
