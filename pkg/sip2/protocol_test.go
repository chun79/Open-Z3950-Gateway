package sip2

import "testing"

func TestComputeChecksum(t *testing.T) {
	// Example from 3M SIP2 Guide
	// "9300CNuser|COpassword|AY0AZ" -> Checksum should be F46D (made up for now, let's verify logic)
	// Let's test basic logic: Sum('A') = 65. 
	// -65 = 0xFFBF. +1 = 0xFFC0? 
	// Wait, standard says: "Binary sum of all characters... take 2's complement"
	// 2's complement of X is (NOT X) + 1. 
	// In Go: (-x) works for 2's complement integer math.
	
	// Test case 1: "9900302.00" (SC Status)
	// Sum: 9+9+0+0+3+0+2+.+0+0 = 57+57+48+48+51+48+50+46+48+48 = 501
	// -501 = 0xFE0B
	// Last 4 hex: FE0B
	
	input := "9900302.00"
	expected := "FE0B"
	got := ComputeChecksum(input)
	if got != expected {
		t.Errorf("ComputeChecksum(%q) = %s; want %s", input, got, expected)
	}
}

func TestBuildMessage(t *testing.T) {
	// Login Message
	// 9300CNlogin|COpass|
	cmd := "93"
	fixed := "00"
	fields := map[string]string{
		"CN": "login",
		"CO": "pass",
	}
	// Note: Map order is random in Go! BuildMessage is unstable for testing unless sorted.
	// For this test we can just check if it ends with AZ and valid checksum.
	
	msg := BuildMessage(cmd, fixed, fields)
	
	// Must contain command
	if msg[:2] != "93" {
		t.Errorf("Wrong command")
	}
	
	// Must end with 4 chars
	if len(msg) < 4 {
		t.Fatalf("Message too short")
	}
	
	// Checksum verification
	// BuildMessage appends "AZxxxx"
	// VerifyChecksum expects the WHOLE string including "AZxxxx"
	// But our VerifyChecksum splits it.
	
	// Let's re-verify the checksum part manually
	dataPart := msg[:len(msg)-4]
	sumPart := msg[len(msg)-4:]
	
	expectedSum := ComputeChecksum(dataPart)
	if sumPart != expectedSum {
		t.Errorf("Checksum mismatch. Data: %s, Got: %s, Calc: %s", dataPart, sumPart, expectedSum)
	}
}
