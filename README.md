# AzDO Build Prometheus Exporter

A Prometheus exporter for Azure DevOps/Azure DevOps Builds. Exposes metrics helpful for reviewing workflow.

- Works with Azure DevOps and Azure DevOps Server.
  - For Server, you will need to set the `ApiVersion="5.0"`.
- Scrapes multiple servers from one exporter, alternatively setup multiple scrapers with their own configs
- Basic support for corporate firewalls
- Configured via TOML

## Azure DevOps Rest Api

This exporter utilises the Rest-Api available against the version of Azure DevOps.
These are set as:

- /_apis/build/builds?apiVersion=6.0
  - For azure DevOps Server version 6 is not available, this can be changed by providing a `ApiVersion="5.0"` in the config.toml for each server connection

## Docker Quickstart

Create a [Personal Access Token](https://docs.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate?view=azure-devops&tabs=preview-page) with the following permissions:

- Worktem (Read)

Create a `config.toml` file with this configuration:

```toml
[servers]
    [servers.azdo]
    address = "https://dev.azure.com/your_org_goes_here"
```

Start the container while binding the `config.toml` created above

```bash
docker run \
  --mount type=bind,source=$(pwd)/config.toml,target=/config.toml \  
  -p 8080:8080 \
  -e TFSEX_azdo_ACCESSTOKEN=your_access_token_goes_here
  --rm \
  ukhydrographicoffice/azdo-build-exporter
```

The collected metrics are available at `:8080/metrics`

## Configuration

The exporter is configured through a [TOML](https://github.com/toml-lang/toml) configuration file, the configuration file location can passed into the exporter through the `--config` flag.

If a `--config` flag isn't provided, the exporter looks for a `config.toml` in its current location.

For each server the exporter scrapes, a configuration "block" is needed. Each of these blocks must have a unique name:

```toml
[servers]
    [servers.unique_name_1]
    address = "https://dev.azure.com/devorg"

    [servers.unique_name_2]
    address = "https://dev.azure.com/devorg2"
```

The unique name is added as a label to the metrics and is useful for differentiating between metrics from different servers. Be careful of changing this unique name as it will force the metrics to have a different label which can cause issues problems, especially on dashboards

Access tokens for Azure DevOps should be provided using environment variables. The required name of the environment variable is in the format `TFSEX_unique_name_1_ACCESSTOKEN`. Access tokens can be added through the configuration file (see [Full Configuration](#Full-Configuration)) but is discouraged.

The default port and url where the metrics are exposed is `:8080/metrics`

The speed of the return is dependant on the number of Tags and the number of Projects, if you wish to pick a selection of projects to pick up, simply add a Projects array to the toml config file (`Projects = ["ProjectName"]).

For Azure DevOps Services, this is an and / or setup, you do not need both, you can use the Tags array property and not the Project array and be fine.

### Basic Configuration

```toml
[servers]
    # On Prem Azure DevOps Server
    [servers.AzDo]
    address = "http://azdo:8080/azdo"
    defaultCollection = "dc"
    ApiVersion = "5.0"

    # Azure DevOps
    [servers.AzureDevOps]
    address = "https://dev.azure.com/devorg"

# As the access tokens aren't specified, the exporter requires them to be set in environment variables:
# TFSEX_AzDo_ACCESSTOKEN
# TFSEX_AzureDevOps_ACCESSTOKEN
# TFSEX_OtherAzureDevOpsInstance_ACCESSTOKEN
```

### Configuration with proxy

```toml
[servers]
    [servers.azuredevops]
    address = "https://dev.azure.com/devorg"
    useProxy = true

[proxy]
    url = "http://proxy.devorg.com:9191"
```

### Full Configuration

```toml
[exporter]
    port = 9595
    endpoint = "/azdometrics"

[servers]
    [servers.azuredevops]
    address = "https://dev.azure.com/devorg"
    useProxy = true
    accessToken = "thisisamadeupaccesstoken"
    # Optional Settings for Azure DevOps Service
    #Project = ["TeamProjectName"]

    [servers.AzDoInstance]
    address = "http://azdo:8080/azdo"
    defaultCollection = "dc"
    # As the access token isn't specified, an environment variable called TFSEX_TFSInstance_ACCESSTOKEN needs to exist
    # Required Settings for Azure DevOps Server
    ApiVersion = "5.0"
    # Optional Settings
    #Project = ["TeamProjectName"]

[proxy]
    url = "http://proxy.devorg.com:9191"
```

## Tips

Set the Prometheus scrape timeout to be larger than 10 seconds as scrapes can sometimes be longer 10s.

## Metrics Exposed

- azdo_build_build_total_scrape_duration_seconds
  - Total time for scrape, Has labels of `name`
- azdo_build_count
  - total builds per project. Has labels of `name, project`
- azdo_build_queue_length_secs_bucket
  - queue length buckets per project, part of histogram. Has labels of `name, project, le`
- azdo_build_queue_length_secs_sum
  - sum of queue length, part of histogram. Has labels of `name,project`
- azdo_build_queue_length_secs_count
  - count of items in queue, part of histogram. Has labels of `name,project`
- azdo_build_queued_count
  - count of builds in queue. Has labels of `name,project`
- azdo_build_running_in_seconds
  - Time build is taking to run in seconds. Has labels of `name, Project, BuildId,BuildNumber, DefinitionId, DefinitionName,results,status`
- azdo_build_running_jobs
  - count of running build job. Has labels of `name, project`
- azdo_build_running_length_secs_bucket
  - running length buckets per project, part of histogram. Has labels of `name, project, le`
- azdo_build_running_length_secs_count
  - count of running builds per project, part of histogram. Has labels of `name, project`
- azdo_build_running_length_secs_sum
  - sum of time for running builds per project, part of histogram. Has labels of `name, project`
- azdo_build_total_length_secs_bucket
  - bucket time for completed builds, part of histogram. Has labels of `name, project, le`
- azdo_build_total_length_secs_sum
  - sum of time for completed builds, part of histogram. Has labels of `name, project`
- azdo_build_total_length_secs_count
  - count of completed builds, part of histogram. Has labels of `name, project`
