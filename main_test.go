package main

import "testing"

func AssertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func AssertOK(t *testing.T, isOK bool) {
	if !isOK {
		t.Fatal("isOk was not true")
	}
}

func AssertEmptyMessage(t *testing.T, msg string) {
	if msg != "" {
		t.Fatal("msg was not empty")
	}
}

func TestUrlCanBeOK(t *testing.T) {
	isOK, msg, err := urlIsOK("https://httpbin.org/")
	AssertNoError(t, err)
	AssertOK(t, isOK)
	AssertEmptyMessage(t, msg)
}

func TestHandlesRedirects(t *testing.T) {
	isOK, msg, err := urlIsOK("https://httpbin.org/redirect/1")
	AssertNoError(t, err)
	AssertOK(t, isOK)
	AssertEmptyMessage(t, msg)
}

func TestHandlesRelativeLinks(t *testing.T) {
}
