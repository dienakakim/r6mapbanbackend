// Package r6mapbanbackend implements a backend for the R6mapban project.
//
// It uses a client-server architecture. S is the central node, connecting
// discrete nodes O, H and B. O represents the Orange Team leader, H represents the mapban host,
// B represents the Blue Team leader, and S represents the backend server.
//
// Here's the workflow:
//
// 1. Phase 0:
//	  H connects to S, S creates sessions for H, O, and B, represented by Base64URL-encoded tokens.
// 	  H gives tokens to O and B so they can connect and participate.
// 2. Phase 1-7:
//	  Ban/pick phases: O ban, B ban, O pick, B pick, O ban, B ban, H decider.
//	  In each phase, the respective side sends a request to S initiating each action.
//	  Phase 7 is also the ending phase that terminates the session. After the host sends
//    the final request, S closes all 3 sessions from H, O, and B, and returns the 3 maps chosen as
//	  the final output for the mapban session.
package r6mapbanbackend
