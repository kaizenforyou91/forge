package compiler

import (
	"errors"
	"testing"
)

func TestDefaultPackageVerificationPolicy(t *testing.T) {
	policy := DefaultPackageVerificationPolicy()

	if !policy.RequireIntegrity {
		t.Fatal("default policy must require integrity")
	}

	if policy.RequireSignature {
		t.Fatal("default policy must not require signature")
	}

	if err := policy.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestStrictPackageVerificationPolicy(t *testing.T) {
	policy := StrictPackageVerificationPolicy()

	if !policy.RequireIntegrity {
		t.Fatal("strict policy must require integrity")
	}

	if !policy.RequireSignature {
		t.Fatal("strict policy must require signature")
	}

	if err := policy.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPackageVerificationPolicyValidate(t *testing.T) {
	testCases := []struct {
		name        string
		policy      PackageVerificationPolicy
		expectError bool
	}{
		{
			name: "default",
			policy: PackageVerificationPolicy{
				RequireIntegrity: true,
				RequireSignature: false,
			},
			expectError: false,
		},
		{
			name: "signature only",
			policy: PackageVerificationPolicy{
				RequireIntegrity: false,
				RequireSignature: true,
			},
			expectError: true,
		},
		{
			name: "both disabled",
			policy: PackageVerificationPolicy{
				RequireIntegrity: false,
				RequireSignature: false,
			},
			expectError: false,
		},
		{
			name: "strict",
			policy: PackageVerificationPolicy{
				RequireIntegrity: true,
				RequireSignature: true,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.policy.Validate()

			if tc.expectError {
				if !errors.Is(err, ErrInvalidVerificationPolicy) {
					t.Fatalf(
						"expected ErrInvalidVerificationPolicy, got %v",
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPackageVerificationPolicyRejectsSignatureWithoutIntegrity(
	t *testing.T,
) {
	policy := PackageVerificationPolicy{
		RequireIntegrity: false,
		RequireSignature: true,
	}

	err := policy.Validate()

	if !errors.Is(err, ErrInvalidVerificationPolicy) {
		t.Fatalf(
			"expected ErrInvalidVerificationPolicy, got %v",
			err,
		)
	}
}
