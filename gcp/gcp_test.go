package gcp

import "testing"

func TestListProjectsRequestUsesOrganizationParent(t *testing.T) {
	req, err := listProjectsRequest("874260368814")
	if err != nil {
		t.Fatalf("listProjectsRequest returned error: %v", err)
	}

	if got, want := req.GetParent(), "organizations/874260368814"; got != want {
		t.Fatalf("parent = %q, want %q", got, want)
	}
}

func TestListProjectsRequestRequiresOrganizationID(t *testing.T) {
	_, err := listProjectsRequest(" ")
	if err == nil {
		t.Fatal("listProjectsRequest returned nil error for empty organization ID")
	}
}

func TestOrganizationParentCanBeUsedForFolderTraversal(t *testing.T) {
	parent, err := organizationParent("874260368814")
	if err != nil {
		t.Fatalf("organizationParent returned error: %v", err)
	}

	if got, want := parent, "organizations/874260368814"; got != want {
		t.Fatalf("parent = %q, want %q", got, want)
	}
}

func TestParseBillingAccountIDsUsesDefaults(t *testing.T) {
	ids, err := parseBillingAccountIDs(defaultBillingAccountIDs)
	if err != nil {
		t.Fatalf("parseBillingAccountIDs returned error: %v", err)
	}
	if len(ids) != 5 {
		t.Fatalf("got %d billing account IDs, want 5", len(ids))
	}
	if _, ok := ids["0165A9-BB7960-BC03A5"]; !ok {
		t.Fatal("default billing account ID was not parsed")
	}
}

func TestParseBillingAccountIDsAllowsEmptyValue(t *testing.T) {
	ids, err := parseBillingAccountIDs("")
	if err != nil {
		t.Fatalf("parseBillingAccountIDs returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("got %d billing account IDs, want empty filter", len(ids))
	}
}

func TestFilterProjectsByID(t *testing.T) {
	projects := []GcpProject{{ID: "included"}, {ID: "excluded"}, {ID: "also-included"}}
	associatedIDs := map[string]struct{}{"included": {}, "also-included": {}}

	filtered := filterProjectsByID(projects, associatedIDs)
	if len(filtered) != 2 {
		t.Fatalf("got %d projects, want 2", len(filtered))
	}
	if filtered[0].ID != "included" || filtered[1].ID != "also-included" {
		t.Fatalf("unexpected filtered projects: %#v", filtered)
	}
}
