package main

import (
	"net/netip"
	"testing"
)

type testCase struct {
	input    string
	expected netip.Prefix
	wantErr  bool
}

func testParsePrefix(t *testing.T, testCases []testCase)
	for _, tc := range testCases {
