package compiler

import "testing"

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
		name   string
		policy PackageVerificationPolicy
	}{
		{
			name: "default",
			policy: PackageVerificationPolicy{
				RequireIntegrity: true,
				RequireSignature: false,
			},
		},
		{
			name: "signature only",
			policy: PackageVerificationPolicy{
				RequireIntegrity: false,
				RequireSignature: true,
			},
		},
		{
			name: "both disabled",
			policy: PackageVerificationPolicy{
				RequireIntegrity: false,
				RequireSignature: false,
			},
		},
		{
			name: "strict",
			policy: PackageVerificationPolicy{
				RequireIntegrity: true,
				RequireSignature: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.policy.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
