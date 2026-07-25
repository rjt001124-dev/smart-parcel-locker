package main

import (
	"testing"

	"github.com/rjt001124-dev/smart-parcel-locker/internal/conf"
)

func TestValidateRuntimeConfigRequiresInternalTokenOutsideTests(t *testing.T) {
	for _, env := range []string{"development", "production"} {
		if err := validateRuntimeConfig(conf.Config{AppEnv: env}); err == nil {
			t.Fatalf("env=%s error=nil, want missing token error", env)
		}
	}
	if err := validateRuntimeConfig(conf.Config{AppEnv: "test"}); err != nil {
		t.Fatalf("test env error=%v", err)
	}
	if err := validateRuntimeConfig(conf.Config{AppEnv: "production", InternalAPIToken: "token"}); err != nil {
		t.Fatalf("configured token error=%v", err)
	}
}
