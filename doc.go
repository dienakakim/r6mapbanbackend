// Package r6mapbanbackend implements a backend for the R6mapban project.
//
// It uses a client-server architecture. S is the central node, connecting
// discrete nodes O, H and B. O represents the Orange Team leader, H represents the mapban host,
// B represents the Blue Team leader, and S represents the backend server.
//
// Here's the workflow:
//
// 1. Phase 0:
//	  H connects to S, S generates tokens for H, O and B to connect and start session.
// 	  H gives tokens to O and B so they can connect and participate.
// 2. Phase 1-7:
//	  Ban/pick phases: O ban, B ban, O pick, B pick, O ban, B ban, H decider.
//	  In each phase, the respective side sends a request to S initiating each action.
// 3. Phase 8:
//	  Results phase, when S frees all sessions. Only the host can access this phase.
package r6mapbanbackend
