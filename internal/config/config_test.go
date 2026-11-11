package config

import (
	"reflect"
	"testing"
)

func TestParseTrustedProxyCIDRs(t *testing.T) {
	got, err := parseTrustedProxyCIDRs(" 127.0.0.1 , 10.0.0.9/8, 127.0.0.1 , ::1/128 ")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"127.0.0.1", "10.0.0.0/8", "::1/128"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("trusted proxy list = %v, want %v", got, want)
	}
	for _, raw := range []string{"invalid", "https://proxy.example", "0.0.0.0/0", "::/0"} {
		if _, err := parseTrustedProxyCIDRs(raw); err == nil {
			t.Errorf("unsafe trusted proxy value %q was accepted", raw)
		}
	}
	empty, err := parseTrustedProxyCIDRs("")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty trusted-proxy configuration should be disabled: %v %v", empty, err)
	}
}
