# AIP Profile Helper

`aiphelper` automates local CLI configuration and Steampipe connection files across AWS, Azure, and GCP multi-cloud environments. For AWS, it creates [AWS CLI profiles](https://docs.aws.amazon.com/cli/v1/userguide/cli-configure-files.html#cli-configure-files-format) for each account and role you can access. For Azure and GCP, it discovers accessible subscriptions or projects and generates matching Steampipe connectors.

Generated configuration is managed between `### AIPHELPER_MARKER_[START|END] ###` blocks, so existing custom configuration outside those markers is preserved.

## Prerequisites & Installation

### Prerequisites

- Go 1.27.1 or later
- Git
- Local cloud authentication credentials for the providers you want to configure

### Building from Source

```bash
# Clone the repository
git clone https://github.com/tamu-edu/aiphelper.git
cd aiphelper

# Build the binary
go build -o aiphelper .

# Optional: install globally
go install
```

## Usage

```text
Usage:
  aiphelper [OPTIONS] <aws | azure | gcp>

Application Options:
  -V, --version       aiphelper Version
  -d, --debug         Enable debug logging
      --kion-url=     Kion URL to use for profile generation (env: KION_URL)
      --kion-apikey=  Kion API token for authentication (env: KION_APIKEY)

Help Options:
  -h, --help  Show this help message

Available commands:
  aws    Initialize AWS
  azure  Initialize Azure
  gcp    Initialize GCP

[aws command options]
      Account Source Options (exactly one required):
          --from-sso     Use AWS Identity Center to get account list
          --from-kion    Use Kion API to get account list

          --sso-start-url=  AWS SSO Start URL (default: https://aggie-innovation-platform.awsapps.com/start)
          --sso-region=     AWS SSO Region (default: us-east-2)
          --sso-role-name=  SSO Role To Assume (must be the same across all accounts) (default: AdministratorAccess)
          --regions=        Comma-separated list of regions to tell Steampipe to connect to (default: uses same search order as aws cli)
          --accounts=       Comma-separated list of accounts to tell Steampipe to connect to (default: all accounts assigned to you through SSO)
          --output-format=  Output format for AWS CLI (default: json)
          --default-region= Default region for AWS CLI operations (default: us-east-1)

[azure command options]
          --tenant-id=       Azure Tenant ID (default: 68f381e3-46da-47b9-ba57-6f322b8f0da1)
      -g, --enum-mgmt-group  Enumerate Azure Management Group descendants for a list of Subscriptions
          --root-group=      management group IDs to begin search for subscriptions (default: tamu)
          --auth-method=     Authentication method to use. Options: [environment, cli, managed-identity, device-code, default] (default: default)

[gcp command options]
      --project-id=             GCP Project ID
      --organization-id=        GCP Organization ID (default: 874260368814)
      --auth-method=            Authentication method (default, service-account, gcloud) (default: default)
      --service-account-key=    Path to service account key file
      --billing-account-ids=    Comma-separated GCP billing account IDs; empty disables filtering (default: 0165A9-BB7960-BC03A5,01C436-796D88-EA0292,0152CF-9A23B4-75E0E7,01030B-9E8B49-9B4A3C,015F25-032066-F3230A)
```

## Cloud Providers

### AWS

`aiphelper` creates an AWS profile for each account and role you have access to based on the account display name and role name. Profile names use the format `<account_name>_<role_name>`, with both values normalized to lowercase, underscores, and a maximum length of 63 characters.

For example, if you have access to `Div Dept My Account 002` with the roles `AdministratorAccess` and `ReadOnlyAccess`, two profiles are created: `div_dept_my_account_002_administratoraccess` and `div_dept_my_account_002_readonlyaccess`.

Profiles are written to the AWS CLI config file, typically `~/.aws/config`. Custom profiles are preserved outside the `### AIPHELPER_MARKER_[START|END] ###` block.

#### Kion Integration

`aiphelper` can generate AWS profiles from Kion that use `kion-cli` to issue short-term credentials transparently. This is useful for directly using the AWS CLI with Kion-managed accounts and with tools that read AWS CLI profiles, such as Steampipe. Kion is the default account source, but you can explicitly select it with `--from-kion`.

To use this feature, `kion-cli` must be installed and configured. See the [Kion CLI documentation](https://github.com/kionsoftware/kion-cli) for more information.

The Kion URL and API key can be set with `--kion-url` and `--kion-apikey`, or with the `KION_URL` and `KION_APIKEY` environment variables. `aiphelper` does not yet support sharing Kion API credentials with `kion-cli`, but this feature is planned for a future release.

#### AWS Identity Center (SSO)

`aiphelper` can generate AWS profiles from AWS Identity Center. AWS Identity Center is not used for AIP customer access, but is still used for some internal and staff accounts. Use the Kion integration if you are an AIP customer.

To use Identity Center as the account source, pass `--from-sso`. This uses the AWS CLI SSO configuration to generate profiles for each account and role you have access to.

Unlike the Kion integration, the SSO integration supports only a single role per account, specified with `--sso-role-name`. Two profiles are created per account in the formats `aws_<account_name>` and `aws_<account_number>`.

For Steampipe, a single aggregate connector named `aws` is created using the `aws_<account_number>` connectors.

If you already have an AWS CLI SSO token that matches the SSO URL and region, it is used. Otherwise, a new device flow authentication is started and the token is cached to disk for later AWS CLI operations.

### Azure

`aiphelper` requires Azure to already be authenticated. By default, it uses the `DefaultAzureCredential` lookup order: environment variables, managed identity, Azure CLI, and other supported default credential sources. To learn more, see [DefaultAzureCredential](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#readme-defaultazurecredential).

The easiest way to get started is to authenticate the Azure CLI with `az login`.

Use `--auth-method` when you need a specific authentication source, such as CLI credentials on a virtual machine that also has a managed identity.

### GCP

`aiphelper` discovers GCP projects under a Google Cloud organization and generates Steampipe configuration in `~/.steampipe/config/gcp.spc`. By default, it enumerates projects under organization `874260368814`. To target a different organization, pass `--organization-id`.

The `gcloud` authentication method uses application default credentials from `~/.config/gcloud/application_default_credentials.json`. To use it, authenticate first with `gcloud auth application-default login`, then run `aiphelper gcp --auth-method=gcloud`.

For service account authentication, pass `--auth-method=service-account --service-account-key=<path-to-key.json>`.

By default, only projects linked to the configured billing account IDs are included. Use `--billing-account-ids=` to include all discovered projects, or provide a comma-separated custom list. Project associations are listed once per billing account to avoid Cloud Billing per-project request quotas. The authenticated identity must have `billing.resourceAssociations.list` on each configured billing account.

## Environment Variables

| Variable | Description |
| --- | --- |
| `KION_URL` | Kion instance URL for AWS account discovery |
| `KION_APIKEY` | Kion API key for authentication |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to a GCP service account key file for Application Default Credentials |

## Examples

### AWS Usage

```bash
# Generate AWS profiles with the default account source and regions
aiphelper aws

# Generate AWS profiles for specific regions
aiphelper aws --regions us-east-1,us-east-2

# AWS CLI command using a generated profile
aws ec2 describe-instances --profile div_dept_my_account_001_readonlyaccess --filters "Name=tag:Environment,Values=test"

# Steampipe query against a single account connector
steampipe query 'select * from aws_div_dept_my_account_001_readonlyaccess.ec2_instance where tags["Environment"] = "test"'

# Steampipe query across all accounts via aggregate role connector
steampipe query 'select * from aws_role_readonlyaccess.ec2_instance where tags["Environment"] = "test"'
```

### GCP Usage

```bash
# Discover projects in the default organization using gcloud application default credentials
aiphelper gcp --auth-method=gcloud

# Discover projects in a specific organization using gcloud application default credentials
aiphelper gcp --organization-id=874260368814 --auth-method=gcloud

# Discover projects in a specific organization using a service account
aiphelper gcp --organization-id=874260368814 --auth-method=service-account --service-account-key=key.json

# Discover all projects without billing-account filtering
aiphelper gcp --billing-account-ids=

# Discover projects linked to a custom billing-account list
aiphelper gcp --billing-account-ids=0165A9-BB7960-BC03A5,01C436-796D88-EA0292

# Query a single project connector in Steampipe
steampipe query 'select name from gcp_my_project.gcp_project'

# Query all discovered projects using the aggregate connector
steampipe query 'select name from gcp_all.gcp_project'
```

## Steampipe

### AWS

`aiphelper` creates one Steampipe connector for each AWS profile for each region specified, defaulting to the AWS CLI default region search order. Connector names use the format `aws_<account_name>_<role_name>`, with account and role names normalized to lowercase, underscores, and a maximum length of 63 characters.

When using Kion as an account source, aggregate connectors are created for each unique role. Connector names use the format `aws_role_<role_name>`. These aggregate connectors allow you to query multiple accounts at once, limited to the accounts that role can access.

For example, if you have access to `Div Dept My Account 001` and `Div Dept My Account 002` with a `ReadOnlyAccess` role, an aggregate connector named `aws_role_readonlyaccess` is created using both `aws_div_dept_my_account_001_readonlyaccess` and `aws_div_dept_my_account_002_readonlyaccess`.

If you use AWS Identity Center as the account source, only one aggregate connector is created, `aws`, using all AWS accounts you have access to.

The Steampipe configuration file is written to `~/.steampipe/config/aws.spc`. Custom connectors and settings are preserved outside the `### AIPHELPER_MARKER_[START|END] ###` block.

### Azure

`aiphelper` creates a Steampipe connector for each Azure subscription it discovers. It also creates an aggregate connector named `azure_all` with every Azure subscription.

### GCP

`aiphelper` creates a Steampipe connector for each GCP project it discovers. It also creates an aggregate connector named `gcp_all` with every discovered project.

### Performance

Limit the number of connectors and tables queried to reduce API calls, especially when using aggregate connectors. Fetch only the precise columns you need from tables instead of selecting every column.

## Troubleshooting

### Authentication Failed

Verify local authentication status with native tools: `az login`, `gcloud auth login`, `gcloud auth application-default login`, or `kion-cli`.

### Missing Accounts, Subscriptions, or Projects

Confirm your identity has the required permissions for discovery, such as `resourcemanager.projects.list` for GCP projects.

For the default billing filter, grant `billing.resourceAssociations.list` on each configured billing account. If this permission is denied, the GCP command cannot apply the billing filter.

### Cloud Billing API has not been used in project before or it is disabled

Enable the Cloud Billing API in your ADC quota project.

`gcloud services enable cloudbilling.googleapis.com --project=your-adc-quota-project-id`

### Verbose Output

Append `-d` or `--debug` to any command to enable detailed logs.
