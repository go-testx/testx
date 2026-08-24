// Package testx is a DX-oriented toolkit layered on top of Go's standard testing package.
//
// It deliberately supports three abstraction levels:
//  1. plain testing, with no testx dependency in the test flow;
//  2. testing + typed Case / Assert / Require primitives;
//  3. declarative Run / RunErr / presets and contract verification.
//
// The higher levels are optional conveniences, not a replacement for testing.T.
package testx
