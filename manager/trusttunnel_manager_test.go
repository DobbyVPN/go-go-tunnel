package manager

import (
	"testing"
)

func TestNewTrustTunnelManager(t *testing.T) {
	manager := NewTrustTunnelManager()
	if manager == nil {
		t.Fatal("NewTrustTunnelManager returned nil")
	}
}

func TestSetProtectSocketCallback(t *testing.T) {
	manager := NewTrustTunnelManager()

	called := false
	var capturedFd int

	manager.SetProtectSocketCallback(func(fd int) int {
		called = true
		capturedFd = fd
		return 0
	})

	// Set as global to test the callback
	SetGlobalManager(manager)

	// Simulate C callback
	result := go_protect_socket(42)

	if !called {
		t.Error("Callback was not called")
	}
	if capturedFd != 42 {
		t.Errorf("Expected fd 42, got %d", capturedFd)
	}
	if result != 0 {
		t.Errorf("Expected result 0, got %d", result)
	}
}

func TestSetStateChangedCallback(t *testing.T) {
	manager := NewTrustTunnelManager()

	called := false
	var capturedState VpnState

	manager.SetStateChangedCallback(func(state VpnState) {
		called = true
		capturedState = state
	})

	// Set as global to test the callback
	SetGlobalManager(manager)

	// Simulate C callback
	go_state_changed(nil, 2)

	if !called {
		t.Error("Callback was not called")
	}
	if capturedState != StateConnected {
		t.Errorf("Expected StateConnected, got %v", capturedState)
	}
}

func TestSetLogCallback(t *testing.T) {
	manager := NewTrustTunnelManager()

	called := false
	var capturedLevel LogLevel
	manager.SetLogCallback(func(level LogLevel, message string) {
		called = true
		capturedLevel = level
	})

	// Set as global to test the callback
	SetGlobalManager(manager)

	// Simulate C callback (pass nil for message since we can't easily create C.char in tests)
	go_log_message(1, nil)

	if !called {
		t.Error("Callback was not called")
	}
	if capturedLevel != LogWarn {
		t.Errorf("Expected LogWarn, got %v", capturedLevel)
	}
}

func TestNilCallbacks(t *testing.T) {
	manager := NewTrustTunnelManager()

	// Don't set any callbacks
	SetGlobalManager(manager)

	// These should not panic
	go_protect_socket(42)
	go_state_changed(nil, 1)
	go_log_message(0, nil)
}

func TestMultipleCallbacks(t *testing.T) {
	manager := NewTrustTunnelManager()

	callCount := 0

	manager.SetProtectSocketCallback(func(fd int) int {
		callCount++
		return 0
	})

	SetGlobalManager(manager)

	// Call multiple times
	go_protect_socket(1)
	go_protect_socket(2)
	go_protect_socket(3)

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestCallbackOverride(t *testing.T) {
	manager := NewTrustTunnelManager()

	firstCalled := false
	secondCalled := false

	manager.SetProtectSocketCallback(func(fd int) int {
		firstCalled = true
		return 0
	})

	manager.SetProtectSocketCallback(func(fd int) int {
		secondCalled = true
		return 0
	})

	SetGlobalManager(manager)
	go_protect_socket(1)

	if firstCalled {
		t.Error("First callback should not have been called")
	}
	if !secondCalled {
		t.Error("Second callback should have been called")
	}
}
