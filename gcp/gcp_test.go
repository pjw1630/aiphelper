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
