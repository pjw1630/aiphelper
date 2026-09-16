package gcp

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	OrganizationID       string `long:"organization-id" description:"GCP Organization ID"`
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

	// Enumerate projects
	projects, err := enumerateProjects(ctx, client)
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

	return resourcemanager.NewProjectsClient(ctx, opts...)
}

func enumerateProjects(ctx context.Context, client *resourcemanager.ProjectsClient) ([]GcpProject, error) {
	var projects []GcpProject

	// List projects
	req := &resourcemanagerpb.ListProjectsRequest{
		Parent: fmt.Sprintf("organizations/%s", options.OrganizationID),
	}

	if options.OrganizationID == "" {
		// If no organization ID, list all accessible projects
		req = &resourcemanagerpb.ListProjectsRequest{}
	}

	it := client.ListProjects(ctx, req)
	for {
		project, err := it.Next()
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

	return projects, nil
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
