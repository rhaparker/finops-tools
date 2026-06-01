package snowflakeoauth

import "testing"

func TestIssuerForEnv(t *testing.T) {
	prod, err := IssuerForEnv("prod")
	if err != nil {
		t.Fatal(err)
	}
	if prod.Issuer != "https://auth.redhat.com/auth/realms/EmployeeIDP" {
		t.Fatalf("prod issuer = %q", prod.Issuer)
	}

	stage, err := IssuerForEnv("stage")
	if err != nil {
		t.Fatal(err)
	}
	if stage.Issuer != "https://auth.stage.redhat.com/auth/realms/EmployeeIDP" {
		t.Fatalf("stage issuer = %q", stage.Issuer)
	}

	if _, err := IssuerForEnv("invalid"); err == nil {
		t.Fatal("expected error")
	}
}
