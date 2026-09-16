# Overview

`aiphelper` automates setting up local CLI configurations and Steampipe connection files across AWS, Azure, and GCP multi-cloud environments. It safely manages block markers (`### AIPHELPER_MARKER_[START|END] ###`) to preserve custom user configurations.

## Prerequisites & Installation

### Prerequisites

* **Go**: 1.27.1 or later
* **Git**
* Local cloud authentication credentials (AWS, Azure, or GCP)

### Building from Source

```bash
# Clone the repository
git clone [https://github.com/tamu-edu/aiphelper.git](https://github.com/tamu-edu/aiphelper.git)
cd aiphelper

# Build the binary
go build -o aiphelper .

# (Optional) Install globally
go install
```

---

## Usage

```bash
aiphelper [GLOBAL OPTIONS] <aws | azure | gcp> [COMMAND OPTIONS]

```

### Global Options

* `-V, --version`: Display application version
* `-d, --debug`: Enable debug logging
* `-h, --help`: Show help text

---

## Cloud Providers

### AWS (`aiphelper aws`)

Generates profiles in `~/.aws/config` formatted as `<account_name>_<role_name>` (normalized to lowercase, underscores, truncated to 63 characters). Also creates matching Steampipe individual and aggregate connectors.

* **Account Sources (Select One)**:
* `--from-kion` *(Default)*: Discovers accounts via the Kion API. Requires `kion-cli` to issue credentials.
* `--from-sso`: Discovers accounts via AWS Identity Center (used primarily for internal/staff accounts). Generates `aws_<account_name>` and `aws_<account_number>` profiles using a single specified role.

* **Command Options**:
* `--from-kion`: Use Kion API to fetch accounts
* `--from-sso`: Use AWS Identity Center to fetch accounts
* `--kion-url`: Kion URL (env: `KION_URL`)
* `--kion-apikey`: Kion API token (env: `KION_APIKEY`)
* `--sso-start-url`: AWS SSO Start URL (default: `https://aggie-innovation-platform.awsapps.com/start`)
* `--sso-region`: AWS SSO Region (default: `us-east-2`)
* `--sso-role-name`: SSO Role to assume across accounts (default: `AdministratorAccess`)
* `--regions`: Comma-separated regions for Steampipe connections
* `--accounts`: Comma-separated list of explicit accounts to include
* `--default-region`: Default AWS CLI region (default: `us-east-1`)
* `--output-format`: AWS CLI output format (default: `json`)

### Azure (`aiphelper azure`)

Enumerates Azure Subscriptions and generates Steampipe connectors. Uses `DefaultAzureCredential` lookup order (Environment, Managed Identity, or Azure CLI) unless specified otherwise.

* **Command Options**:
* `--tenant-id`: Azure Tenant ID (default: `68f381e3-46da-47b9-ba57-6f322b8f0da1`)
* `-g, --enum-mgmt-group`: Enumerate Management Group descendants to discover Subscriptions
* `--root-group`: Management Group ID to begin search (default: `tamu`)
* `--auth-method`: Auth strategy (`default`, `environment`, `cli`, `managed-identity`, `device-code`)

### GCP (`aiphelper gcp`)

Discovers GCP projects and generates Steampipe configurations in `~/.steampipe/config/gcp.spc`. Creates individual `gcp_<normalized_project_name>` connectors and a `gcp_all` aggregate connector.

* **Command Options**:
* `--project-id`: Target specific GCP Project ID
* `--organization-id`: Filter projects by GCP Organization ID
* `--auth-method`: Authentication strategy (`default`, `service-account`, `gcloud`)
* `--service-account-key`: Path to Service Account JSON key file

---

## Environment Variables

| Variable | Description |
| --- | --- |
| `KION_URL` | Kion instance URL for AWS account discovery |
| `KION_APIKEY` | Kion API key for authentication |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to GCP Service Account key file |

---

## Examples

### AWS Usage

```bash
# Generate AWS profiles for specific regions using Kion
aiphelper aws --regions us-east-1,us-east-2

# AWS CLI command using a generated profile
aws ec2 describe-instances --profile div_dept_my_account_001_readonlyaccess

# Steampipe query against a single account connector
steampipe query 'select * from aws_div_dept_my_account_001_readonlyaccess.ec2_instance'

# Steampipe query across all accounts via aggregate role connector
steampipe query 'select * from aws_role_readonlyaccess.ec2_instance'

```

### GCP Usage

```bash
# Discover projects in a specific organization using gcloud login
aiphelper gcp --organization-id=874260368814 --auth-method=gcloud

# Discover projects in a specific organization using a Service Account
aiphelper gcp --organization-id=874260368814 --auth-method=service-account --service-account-key=key.json

# Query a single project connector in Steampipe
steampipe query 'select * from gcp_my_project.compute_instance'

# Query all projects using the aggregate connector
steampipe query 'select * from gcp_all.compute_instance'

```

---

## Troubleshooting

* **Authentication Failed**: Verify local authentication status using native tools (`az login`, `gcloud auth login` and `gcloud auth application-default login`, or `kion-cli`).
* **Missing Accounts/Projects**: Confirm your identity holds necessary permission scopes (e.g., `resourcemanager.projects.list` for GCP).
* **Verbose Output**: Append the `-d` or `--debug` flag to any command to review detailed logs.
