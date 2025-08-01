package hyperdrive

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Hardware Transactional Memory (HTM) support
// Intel TSX (Transactional Synchronization Extensions)
// ARM TME (Transactional Memory Extension)

// Transaction represents a hardware transaction
type Transaction struct {
	id        uint64
	status    uint32
	retries   uint32
	maxRetries uint32
	fallback  func() error
	stats     *TMStats
}

// TMStats tracks transactional memory statistics
type TMStats struct {
	Started   atomic.Uint64
	Committed atomic.Uint64
	Aborted   atomic.Uint64
	Retries   atomic.Uint64
	Fallbacks atomic.Uint64
	Conflicts atomic.Uint64
	Capacity  atomic.Uint64
	Debug     atomic.Uint64
}

// TransactionalMap is a map that uses HTM for concurrency
type TransactionalMap struct {
	buckets    []TMBucket
	bucketMask uint32
	stats      TMStats
	useHTM     bool
}

// TMBucket represents a bucket in the transactional map
type TMBucket struct {
	entries []TMEntry
	lock    sync.Mutex // Fallback lock
	version atomic.Uint64
}

// TMEntry represents an entry in the map
type TMEntry struct {
	key   uint64
	value unsafe.Pointer
	next  *TMEntry
}

// HTM status codes
const (
	TM_STARTED    = 0
	TM_ABORT_CONFLICT = 1 << iota
	TM_ABORT_CAPACITY
	TM_ABORT_DEBUG
	TM_ABORT_NESTED
	TM_ABORT_RETRY
	TM_ABORT_USER
)

// HTM configuration
const (
	TM_MAX_RETRIES = 5
	TM_BUCKET_SIZE = 16
	TM_MAP_BUCKETS = 1024
)

var (
	htmSupported bool
	htmCheckOnce sync.Once
	globalTMStats TMStats
)

// checkHTMSupport checks if hardware transactional memory is available
func checkHTMSupport() bool {
	htmCheckOnce.Do(func() {
		if runtime.GOARCH == "amd64" {
			// Check for Intel TSX support
			_, _, ecx, _ := cpuid(7, 0)
			htmSupported = (ecx & (1 << 11)) != 0 // RTM bit
		} else if runtime.GOARCH == "arm64" {
			// Check for ARM TME support
			// Would check ID_AA64ISAR0_EL1.TME
			htmSupported = false // Not widely available yet
		}
	})
	return htmSupported
}

// NewTransactionalMap creates a new transactional map
func NewTransactionalMap(buckets uint32) *TransactionalMap {
	if buckets == 0 {
		buckets = TM_MAP_BUCKETS
	}

	// Ensure power of 2
	buckets = nextPowerOf2(buckets)

	tm := &TransactionalMap{
		buckets:    make([]TMBucket, buckets),
		bucketMask: buckets - 1,
		useHTM:     checkHTMSupport(),
	}

	for i := range tm.buckets {
		tm.buckets[i].entries = make([]TMEntry, 0, TM_BUCKET_SIZE)
	}

	return tm
}

// Get retrieves a value using HTM
func (tm *TransactionalMap) Get(key uint64) (unsafe.Pointer, bool) {
	bucketIdx := key & uint64(tm.bucketMask)
	bucket := &tm.buckets[bucketIdx]

	if tm.useHTM {
		return tm.getHTM(bucket, key)
	}
	return tm.getFallback(bucket, key)
}

// getHTM uses hardware transactional memory for get
func (tm *TransactionalMap) getHTM(bucket *TMBucket, key uint64) (unsafe.Pointer, bool) {
	var value unsafe.Pointer
	var found bool

	tx := &Transaction{
		maxRetries: TM_MAX_RETRIES,
		stats:      &tm.stats,
	}

	err := tm.executeTransaction(tx, func() error {
		// Transaction body
		for i := range bucket.entries {
			entry := &bucket.entries[i]
			if entry.key == key {
				value = entry.value
				found = true
				return nil
			}
		}
		return nil
	})

	if err != nil {
		// Fallback to lock-based
		return tm.getFallback(bucket, key)
	}

	return value, found
}

// getFallback uses traditional locking
func (tm *TransactionalMap) getFallback(bucket *TMBucket, key uint64) (unsafe.Pointer, bool) {
	bucket.lock.Lock()
	defer bucket.lock.Unlock()

	for i := range bucket.entries {
		entry := &bucket.entries[i]
		if entry.key == key {
			return entry.value, true
		}
	}
	return nil, false
}

// Put inserts or updates a value using HTM
func (tm *TransactionalMap) Put(key uint64, value unsafe.Pointer) error {
	bucketIdx := key & uint64(tm.bucketMask)
	bucket := &tm.buckets[bucketIdx]

	if tm.useHTM {
		return tm.putHTM(bucket, key, value)
	}
	return tm.putFallback(bucket, key, value)
}

// putHTM uses hardware transactional memory for put
func (tm *TransactionalMap) putHTM(bucket *TMBucket, key uint64, value unsafe.Pointer) error {
	tx := &Transaction{
		maxRetries: TM_MAX_RETRIES,
		stats:      &tm.stats,
	}

	return tm.executeTransaction(tx, func() error {
		// Transaction body
		for i := range bucket.entries {
			entry := &bucket.entries[i]
			if entry.key == key {
				// Update existing
				entry.value = value
				return nil
			}
		}

		// Add new entry
		if len(bucket.entries) < cap(bucket.entries) {
			bucket.entries = append(bucket.entries, TMEntry{
				key:   key,
				value: value,
			})
			return nil
		}

		// Bucket full, need to resize
		return errors.New("bucket full")
	})
}

// putFallback uses traditional locking
func (tm *TransactionalMap) putFallback(bucket *TMBucket, key uint64, value unsafe.Pointer) error {
	bucket.lock.Lock()
	defer bucket.lock.Unlock()

	for i := range bucket.entries {
		entry := &bucket.entries[i]
		if entry.key == key {
			entry.value = value
			return nil
		}
	}

	if len(bucket.entries) < cap(bucket.entries) {
		bucket.entries = append(bucket.entries, TMEntry{
			key:   key,
			value: value,
		})
		return nil
	}

	return errors.New("bucket full")
}

// Delete removes a value using HTM
func (tm *TransactionalMap) Delete(key uint64) bool {
	bucketIdx := key & uint64(tm.bucketMask)
	bucket := &tm.buckets[bucketIdx]

	if tm.useHTM {
		return tm.deleteHTM(bucket, key)
	}
	return tm.deleteFallback(bucket, key)
}

// deleteHTM uses hardware transactional memory for delete
func (tm *TransactionalMap) deleteHTM(bucket *TMBucket, key uint64) bool {
	deleted := false

	tx := &Transaction{
		maxRetries: TM_MAX_RETRIES,
		stats:      &tm.stats,
	}

	err := tm.executeTransaction(tx, func() error {
		for i := range bucket.entries {
			entry := &bucket.entries[i]
			if entry.key == key {
				// Remove by swapping with last
				last := len(bucket.entries) - 1
				if i < last {
					bucket.entries[i] = bucket.entries[last]
				}
				bucket.entries = bucket.entries[:last]
				deleted = true
				return nil
			}
		}
		return nil
	})

	if err != nil {
		return tm.deleteFallback(bucket, key)
	}

	return deleted
}

// deleteFallback uses traditional locking
func (tm *TransactionalMap) deleteFallback(bucket *TMBucket, key uint64) bool {
	bucket.lock.Lock()
	defer bucket.lock.Unlock()

	for i := range bucket.entries {
		entry := &bucket.entries[i]
		if entry.key == key {
			last := len(bucket.entries) - 1
			if i < last {
				bucket.entries[i] = bucket.entries[last]
			}
			bucket.entries = bucket.entries[:last]
			return true
		}
	}
	return false
}

// executeTransaction executes a hardware transaction
func (tm *TransactionalMap) executeTransaction(tx *Transaction, fn func() error) error {
	for retry := uint32(0); retry < tx.maxRetries; retry++ {
		tx.stats.Started.Add(1)

		// Start transaction
		status := tmBegin()
		if status == TM_STARTED {
			// Execute transaction body
			err := fn()
			if err != nil {
				tmAbort(TM_ABORT_USER)
				return err
			}

			// Commit transaction
			if tmCommit() {
				tx.stats.Committed.Add(1)
				return nil
			}
		}

		// Transaction aborted
		tx.stats.Aborted.Add(1)
		tx.stats.Retries.Add(1)

		// Check abort reason
		if status&TM_ABORT_CONFLICT != 0 {
			tx.stats.Conflicts.Add(1)
		} else if status&TM_ABORT_CAPACITY != 0 {
			tx.stats.Capacity.Add(1)
			break // No point retrying capacity aborts
		} else if status&TM_ABORT_DEBUG != 0 {
			tx.stats.Debug.Add(1)
		}

		// Exponential backoff
		backoff(retry)
	}

	// All retries failed, use fallback
	tx.stats.Fallbacks.Add(1)
	return errors.New("transaction failed")
}

// HTM primitives (would be implemented in assembly)

// tmBegin starts a hardware transaction
func tmBegin() uint32 {
	// On Intel: XBEGIN
	// On ARM: TSTART
	// For simulation, always succeed
	if htmSupported {
		return TM_STARTED
	}
	return TM_ABORT_USER
}

// tmCommit commits a hardware transaction
func tmCommit() bool {
	// On Intel: XEND
	// On ARM: TCOMMIT
	return htmSupported
}

// tmAbort aborts a hardware transaction
func tmAbort(code uint32) {
	// On Intel: XABORT
	// On ARM: TCANCEL
}

// tmTest tests if in transaction
func tmTest() bool {
	// On Intel: XTEST
	// On ARM: Check TACTIVE
	return false
}

// Optimistic concurrency control using HTM

// OptimisticLock provides optimistic locking with HTM
type OptimisticLock struct {
	version atomic.Uint64
	fallback sync.RWMutex
	useHTM  bool
	stats   TMStats
}

// NewOptimisticLock creates a new optimistic lock
func NewOptimisticLock() *OptimisticLock {
	return &OptimisticLock{
		useHTM: checkHTMSupport(),
	}
}

// OptimisticRead performs an optimistic read
func (ol *OptimisticLock) OptimisticRead(fn func() error) error {
	if !ol.useHTM {
		ol.fallback.RLock()
		defer ol.fallback.RUnlock()
		return fn()
	}

	// Read version before transaction
	versionBefore := ol.version.Load()

	// Execute in transaction
	tx := &Transaction{
		maxRetries: TM_MAX_RETRIES,
		stats:      &ol.stats,
	}

	err := executeTransactionWithVersion(tx, &ol.version, versionBefore, fn)
	if err != nil {
		// Fallback to read lock
		ol.fallback.RLock()
		defer ol.fallback.RUnlock()
		return fn()
	}

	return nil
}

// OptimisticWrite performs an optimistic write
func (ol *OptimisticLock) OptimisticWrite(fn func() error) error {
	if !ol.useHTM {
		ol.fallback.Lock()
		defer ol.fallback.Unlock()
		return fn()
	}

	// Increment version in transaction
	tx := &Transaction{
		maxRetries: TM_MAX_RETRIES,
		stats:      &ol.stats,
	}

	err := executeTransactionWithVersionUpdate(tx, &ol.version, fn)
	if err != nil {
		// Fallback to write lock
		ol.fallback.Lock()
		defer ol.fallback.Unlock()
		ol.version.Add(1)
		return fn()
	}

	return nil
}

// executeTransactionWithVersion executes transaction with version check
func executeTransactionWithVersion(tx *Transaction, version *atomic.Uint64, expected uint64, fn func() error) error {
	return executeTransaction(tx, func() error {
		// Check version hasn't changed
		if version.Load() != expected {
			tmAbort(TM_ABORT_CONFLICT)
			return errors.New("version mismatch")
		}
		return fn()
	})
}

// executeTransactionWithVersionUpdate executes transaction with version update
func executeTransactionWithVersionUpdate(tx *Transaction, version *atomic.Uint64, fn func() error) error {
	return executeTransaction(tx, func() error {
		err := fn()
		if err != nil {
			return err
		}
		// Update version on success
		version.Add(1)
		return nil
	})
}

// executeTransaction is a helper that wraps the TM's method
func executeTransaction(tx *Transaction, fn func() error) error {
	// This is a simplified version - real implementation would be in assembly
	for retry := uint32(0); retry < tx.maxRetries; retry++ {
		tx.stats.Started.Add(1)

		if htmSupported {
			// Simulate successful transaction
			err := fn()
			if err == nil {
				tx.stats.Committed.Add(1)
				return nil
			}
		}

		tx.stats.Aborted.Add(1)
		backoff(retry)
	}

	tx.stats.Fallbacks.Add(1)
	return errors.New("transaction failed")
}

// backoff implements exponential backoff
func backoff(retry uint32) {
	if retry > 0 {
		// Simple exponential backoff
		for i := uint32(0); i < (1 << retry); i++ {
			runtime.Gosched()
		}
	}
}

// GetStats returns HTM statistics
func (tm *TransactionalMap) GetStats() TMStats {
	return tm.stats
}

// GetGlobalStats returns global HTM statistics
func GetGlobalStats() TMStats {
	return globalTMStats
}

// nextPowerOf2 returns the next power of 2
func nextPowerOf2(n uint32) uint32 {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}