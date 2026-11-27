package controllers

import "testing"

// TestTMDControllerIsBuiltAfterDBConnect guards against the package-init trap:
// building the controller in a package-level initializer captures postgres.DB
// while it is still nil, so every DB-backed client TMD endpoint panics into a
// bare 500. The controller must stay unbuilt until InitTaskManagementController
// runs after postgres.ConnectDatabase().
func TestTMDControllerIsBuiltAfterDBConnect(t *testing.T) {
	if defaultTMDController == nil {
		t.Fatal("defaultTMDController must be non-nil after package init")
	}
	if defaultTMDController.service != nil {
		t.Fatal("defaultTMDController was built at package init, before postgres.ConnectDatabase()")
	}

	InitTaskManagementController()

	if defaultTMDController.service == nil {
		t.Fatal("InitTaskManagementController did not build the service")
	}
}
