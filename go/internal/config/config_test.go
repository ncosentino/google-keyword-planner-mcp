package config

import "testing"

func TestDeveloperTokenIsOptionalButOAuthAndCustomerRemainRequired(t *testing.T) {
	complete := Config{ClientID: "id", ClientSecret: "secret", RefreshToken: "refresh", CustomerID: "123"}
	if !complete.IsComplete() {
		t.Fatal("configuration without a developer token should be complete")
	}
	for _, field := range []string{"client", "secret", "refresh", "customer"} {
		config := complete
		switch field {
		case "client":
			config.ClientID = ""
		case "secret":
			config.ClientSecret = ""
		case "refresh":
			config.RefreshToken = ""
		case "customer":
			config.CustomerID = ""
		}
		if config.IsComplete() {
			t.Errorf("configuration missing %s accepted", field)
		}
	}
}
