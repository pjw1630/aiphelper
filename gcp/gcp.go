package gcp

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	_ "embed"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/tamu-edu/aiphelper/utils"
)

var (
	steampipeTemplate *template.Template
)

//go:embed steampipe.gospc
var steampipeTemplateString string

type Options struct {
	ProjectID            string `long:"project-id" description:"GCP Project ID"`
	OrganizationID       string `long:"organization-id" default:"874260368814" description:"GCP Organization ID"`
	AuthenticationMethod string `long:"auth-method" default:"default" description:"Authentication method (default, service-account, gcloud)"`
	ServiceAccountKey    string `long:"service-account-key" description:"Path to service account key file"`
}

type GcpProject struct {
	ID             string
	Name           string
	DisplayName    string
	NormalizedName string
}

type TemplateData struct {
	Marker            string
	Projects          []GcpProject
	AggregationString string
}

func Init() {
	var err error
	steampipeTemplate = template.Must(template.New("steampipeTemplate").Parse(steampipeTemplateString))

	// Initialize GCP client
	ctx := context.Background()
	client, err := newProjectsClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create GCP client: %v", err)
	}
	defer client.Close()
	foldersClient, err := newFoldersClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create GCP folders client: %v", err)
	}
	defer foldersClient.Close()

	// Enumerate projects
	projects, err := enumerateProjects(ctx, client, foldersClient)
	if err != nil {
		log.Fatalf("Failed to enumerate projects: %v", err)
	}

	if len(projects) == 0 {
		log.Fatal("No GCP projects found")
	}

	// Generate Steampipe configuration
	err = updateSteampipeGcpConfigFile(projects)
	if err != nil {
		log.Fatalf("Failed to update Steampipe config: %v", err)
	}

	fmt.Printf("Successfully generated Steampipe configuration for %d GCP projects\n", len(projects))
}

func newProjectsClient(ctx context.Context) (*resourcemanager.ProjectsClient, error) {
	opts, err := clientOptions(ctx)
	if err != nil {
		return nil, err
	}

	return resourcemanager.NewProjectsClient(ctx, opts...)
}

func newFoldersClient(ctx context.Context) (*resourcemanager.FoldersClient, error) {
	opts, err := clientOptions(ctx)
	if err != nil {
		return nil, err
	}

	return resourcemanager.NewFoldersClient(ctx, opts...)
}

func clientOptions(ctx context.Context) ([]option.ClientOption, error) {
	var opts []option.ClientOption

	switch options.AuthenticationMethod {
	case "service-account":
		if options.ServiceAccountKey == "" {
			return nil, fmt.Errorf("service account key file path is required for service-account authentication")
		}
		opts = append(opts, option.WithCredentialsFile(options.ServiceAccountKey))
	case "gcloud":
		// Use gcloud application default credentials
		opts = append(opts, option.WithCredentialsFile(filepath.Join(os.Getenv("HOME"), ".config", "gcloud", "application_default_credentials.json")))
	case "default":
		// Use Application Default Credentials
		credentials, err := google.FindDefaultCredentials(ctx, resourcemanager.DefaultAuthScopes()...)
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %v", err)
		}
		opts = append(opts, option.WithCredentials(credentials))
	}

	return opts, nil
}

func enumerateProjects(ctx context.Context, projectsClient *resourcemanager.ProjectsClient, foldersClient *resourcemanager.FoldersClient) ([]GcpProject, error) {
	var projects []GcpProject

	parent, err := organizationParent(options.OrganizationID)
	if err != nil {
		return nil, err
	}

	return enumerateProjectsUnderParent(ctx, projectsClient, foldersClient, parent, projects)
}

func enumerateProjectsUnderParent(ctx context.Context, projectsClient *resourcemanager.ProjectsClient, foldersClient *resourcemanager.FoldersClient, parent string, projects []GcpProject) ([]GcpProject, error) {
	projectIt := projectsClient.ListProjects(ctx, &resourcemanagerpb.ListProjectsRequest{Parent: parent})
	for {
		project, err := projectIt.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate projects: %v", err)
		}

		// Skip deleted projects (commented out as Project_DELETED is not available in current API)
		// if project.GetState() == resourcemanagerpb.Project_DELETED {
		// 	continue
		// }

		normalizedName := utils.SnakeCase(project.GetName())
		if normalizedName == "" {
			normalizedName = utils.SnakeCase(project.GetProjectId())
		}

		projects = append(projects, GcpProject{
			ID:             project.GetProjectId(),
			Name:           project.GetName(),
			DisplayName:    project.GetDisplayName(),
			NormalizedName: normalizedName,
		})
	}

	folderIt := foldersClient.ListFolders(ctx, &resourcemanagerpb.ListFoldersRequest{Parent: parent})
	for {
		folder, err := folderIt.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate folders under %s: %v", parent, err)
		}

		projects, err = enumerateProjectsUnderParent(ctx, projectsClient, foldersClient, folder.GetName(), projects)
		if err != nil {
			return nil, err
		}
	}

	return projects, nil
}

func listProjectsRequest(organizationID string) (*resourcemanagerpb.ListProjectsRequest, error) {
	parent, err := organizationParent(organizationID)
	if err != nil {
		return nil, err
	}

	return &resourcemanagerpb.ListProjectsRequest{Parent: parent}, nil
}

func organizationParent(organizationID string) (string, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return "", fmt.Errorf("--organization-id is required")
	}

	return fmt.Sprintf("organizations/%s", organizationID), nil
}

func updateSteampipeGcpConfigFile(projects []GcpProject) error {
	// Create aggregation string
	var aggregationParts []string
	for _, project := range projects {
		aggregationParts = append(aggregationParts, fmt.Sprintf("\"gcp_%s\"", project.NormalizedName))
	}

	data := TemplateData{
		Marker:            "AIPHELPER_MARKER",
		Projects:          projects,
		AggregationString: utils.JoinStrings(aggregationParts, ", "),
	}

	// Generate the configuration content
	var configContent bytes.Buffer
	err := steampipeTemplate.Execute(&configContent, data)
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	// Write to file using marker-based replacement (matching AWS/Azure pattern)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	outputPath := filepath.Join(homeDir, ".steampipe", "config", "gcp.spc")

	// Use the same marker-based file replacement as AWS/Azure to preserve existing content
	err = utils.CreateOrReplaceInFile(outputPath, configContent.String())
	if err != nil {
		return fmt.Errorf("failed to write Steampipe config: %v", err)
	}

	return nil
}
