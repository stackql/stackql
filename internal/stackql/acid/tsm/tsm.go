package tsm

// Transaction Storage Manager (TSM)
// A monolith containing:
//  1. Log Manager.  Write Ahead Logging; WAL.
//  2. Lock Manager.  2 Phase Locking; 2PL... **not** to be confused with 2 Phase Commit.
//  3. Access Methods.  In a conventional RDBMS B+-tree index and heap file access primitives, including latches.
//  4. Buffer Pool.  It is possible to externalise this componsnes as it does not require shared internal
//     awareness with other components (as oppose to the other 3 components).
type TSM interface {
	//
}
