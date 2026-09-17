# AIP Profile Helper (`aiphelper`)

`aiphelper` automates local CLI configurations and Steampipe connection files across AWS, Azure, and GCP multi-cloud environments.

It generates AWS CLI profiles for accessible accounts and roles, discovers Azure subscriptions and GCP projects, and outputs matching Steampipe connectors. All generated configurations are isolated within `### AIPHELPER_MARKER_[START|END] ###` blocks to preserve your existing custom settings.

---

## Features

* **AWS CLI & Steampipe Management:** Automatically generates profile names (`<account>_<role>`) and writes them to `~/.aws/config` and `~/.steampipe/config/aws.spc`.
* **Kion & AWS SSO Integration:** Integrates with Kion API/CLI for short-term credential generation and AWS Identity Center (SSO).
* **Multi-Cloud Discovery:** Auto-discovers Azure Subscriptions and GCP Projects under specified organizations/billing accounts.
* **Aggregate Connectors:** Generates multi-account Steampipe aggregate connectors (`aws_role_<role>`, `azure_all`, `gcp_all`) for cross-environment querying.
* **Non-Destructive:** Safeguards existing local configuration outside marker blocks.

---

## Prerequisites & Installation

### Prerequisites

* **Go:** 1.27.1 or higher
* **Git**
* Authenticated local CLI tools (`kion-cli`, `az`, `gcloud`) depending on the providers used.

### Installation

```bash
# Clone the repository
git clone https://github.com/tamu-edu/aiphelper.git
cd aiphelper

# Build the binary
go build -o aiphelper .

# Optional: Install globally to $GOPATH/bin
go install

```

---

## Usage Syntax

```bash
aiphelper [GLOBAL OPTIONS] <aws | azure | gcp> [COMMAND OPTIONS]

```

### Global Options

| Flag | Environment Variable | Description |
| --- | --- | --- |
| `-V, --version` | — | Display application version |
| `-d, --debug` | — | Enable verbose debug logging |
| `--kion-url` | `KION_URL` | Base URL for Kion API |
| `--kion-apikey` | `KION_APIKEY` | Authentication token for Kion API |
| `-h, --help` | — | Show help message |

---

## Provider Setup

### AWS

Discovers AWS accounts via Kion (default) or AWS Identity Center (SSO) and updates `~/.aws/config` and `~/.steampipe/config/aws.spc`. Profile names follow the format `<account_name>_<role_name>` (normalized to lowercase, max 63 characters).

#### Options

* `--from-kion`: Use Kion API (default). Requires `kion-cli`.
* `--from-sso`: Use AWS Identity Center.
* `--sso-start-url`: AWS SSO Start URL *(Default: `[https://aggie-innovation-platform.awsapps.com/start](https://aggie-innovation-platform.awsapps.com/start)`)*.
* `--sso-region`: AWS SSO Region *(Default: `us-east-2`)*.
* `--sso-role-name`: Role to assume across accounts *(Default: `AdministratorAccess`)*.
* `--regions`: Comma-separated list of target regions for Steampipe.
* `--accounts`: Comma-separated list of account IDs to target.
* `--output-format`: Output format for AWS CLI *(Default: `json`)*.
* `--default-region`: Default AWS region *(Default: `us-east-1`)*.

#### Examples

```bash
# Generate AWS profiles using default settings (Kion)
aiphelper aws

# Generate AWS profiles for specific regions
aiphelper aws --regions us-east-1,us-east-2

# Generate AWS profiles using Identity Center (SSO)
aiphelper aws --from-sso --sso-role-name ReadOnlyAccess

# Query AWS CLI using generated profile
aws ec2 describe-instances --profile div_dept_my_account_001_readonlyaccess

# Steampipe: Query single account profile
steampipe query 'select * from aws_div_dept_my_account_001_readonlyaccess.ec2_instance'

# Steampipe: Query across all accounts via aggregate role connector
steampipe query 'select * from aws_role_readonlyaccess.ec2_instance'

```

---

### Azure

Discovers accessible Azure Subscriptions and generates `~/.steampipe/config/azure.spc` along with an aggregate connector `azure_all`.

#### Options

* `--tenant-id`: Azure Tenant ID *(Default: `68f381e3-46da-47b9-ba57-6f322b8f0da1`)*.
* `-g, --enum-mgmt-group`: Enumerate Management Group descendants for subscriptions.
* `--root-group`: Root Management Group ID to begin search *(Default: `tamu`)*.
* `--auth-method`: Authentication method (`environment`, `cli`, `managed-identity`, `device-code`, `default`).

#### Examples

```bash
# Authenticate CLI first
az login

# Generate Azure Steampipe configuration
aiphelper azure

# Query all subscriptions in Steampipe
steampipe query 'select name, subscription_id from azure_all.azure_subscription'

```

---

### GCP

Discovers GCP projects under an organization, filtering by billing accounts where required, and writes configuration to `~/.steampipe/config/gcp.spc`.

#### Options

* `--organization-id`: GCP Organization ID *(Default: `874260368814`)*.
* `--project-id`: Target specific GCP Project ID.
* `--auth-method`: Auth method: `default`, `service-account`, or `gcloud` *(Default: `default`)*.
* `--service-account-key`: Path to service account JSON key file.
* `--billing-account-ids`: Comma-separated billing IDs to filter projects. Pass empty string (`""`) to disable filtering.

#### Examples

```bash
# Authenticate via Application Default Credentials
gcloud auth application-default login

# Discover projects using gcloud ADC
aiphelper gcp --auth-method=gcloud

# Discover using a service account key
aiphelper gcp --auth-method=service-account --service-account-key=/path/to/key.json

# Disable billing filtering to fetch all accessible projects
aiphelper gcp --billing-account-ids=""

# Steampipe: Query a specific project
steampipe query 'select name from gcp_my_project.gcp_project'

# Steampipe: Query all projects via aggregate connector
steampipe query 'select name from gcp_all.gcp_project'

```

---

## Environment Variables

| Variable | Description |
| --- | --- |
| `KION_URL` | Base URL for the Kion API endpoint |
| `KION_APIKEY` | API Key for authenticating with Kion |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to GCP service account JSON key |

---

## Steampipe Connector Structures

| Cloud Provider | Individual Connector | Aggregate Connector | Config Output Location |
| --- | --- | --- | --- |
| **AWS** | `aws_<account_name>_<role>` | `aws_role_<role_name>` (Kion) or `aws` (SSO) | `~/.steampipe/config/aws.spc` |
| **Azure** | `azure_<subscription_name>` | `azure_all` | `~/.steampipe/config/azure.spc` |
| **GCP** | `gcp_<project_id>` | `gcp_all` | `~/.steampipe/config/gcp.spc` |

*Tip: For optimal query performance when using aggregate connectors (`aws_role_*`, `azure_all`, `gcp_all`), explicitly define needed columns instead of using `SELECT *`.*

---

## Troubleshooting

* **Authentication Errors:** Ensure local CLI credentials are active prior to running `aiphelper` (`az login`, `gcloud auth application-default login`, or `kion-cli`).
* **Missing GCP Projects:** Verify your identity has `resourcemanager.projects.list` on the organization and `billing.resourceAssociations.list` on target billing accounts.
* **GCP Cloud Billing API Disabled:** Enable the billing API in your Application Default Credentials (ADC) quota project:

```bash
gcloud services enable cloudbilling.googleapis.com --project=YOUR_ADC_PROJECT_ID
```

* **Debug Logging:** Append `-d` or `--debug` to any command for detailed diagnostic output.