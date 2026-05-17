package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePort_Valid(t *testing.T) {
	p, err := parsePort("8080")
	assert.NoError(t, err)
	assert.Equal(t, 8080, p)
}

func TestParsePort_Zero(t *testing.T) {
	_, err := parsePort("0")
	assert.Error(t, err)
}

func TestParsePort_Negative(t *testing.T) {
	_, err := parsePort("-1")
	assert.Error(t, err)
}

func TestParsePort_OutOfRange(t *testing.T) {
	_, err := parsePort("70000")
	assert.Error(t, err)
}

func TestParsePort_NonNumeric(t *testing.T) {
	_, err := parsePort("abc")
	assert.Error(t, err)
}

func TestParsePort_BoundaryLow(t *testing.T) {
	p, err := parsePort("1")
	assert.NoError(t, err)
	assert.Equal(t, 1, p)
}

func TestParsePort_BoundaryHigh(t *testing.T) {
	p, err := parsePort("65535")
	assert.NoError(t, err)
	assert.Equal(t, 65535, p)
}
