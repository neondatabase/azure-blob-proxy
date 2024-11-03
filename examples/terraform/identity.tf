# See https://learn.microsoft.com/en-us/azure/aks/workload-identity-overview?tabs=dotnet#service-account-labels-and-annotations

# NOTE: AKS azurerm_kubernetes_cluster.example is not the topic of this example

## k8s service account
resource "kubernetes_service_account" "example" {
  metadata {
    annotations = {
      "azure.workload.identity/client-id" = azurerm_user_assigned_identity.example.client_id
    }
    name      = "azure-blob-proxy-demo"
    namespace = "default"
  }
}

## User assigned identity
resource "azurerm_user_assigned_identity" "example" {
  name                = "azure-blob-proxy-demo"
  resource_group_name = azurerm_resource_group.example.name
  location            = azurerm_resource_group.example.location
}

## Azure AD federated identity used to federate kubernetes with Azure AD
resource "azurerm_federated_identity_credential" "example" {
  name                = azurerm_user_assigned_identity.example.name
  resource_group_name = azurerm_user_assigned_identity.example.resource_group_name
  parent_id           = azurerm_user_assigned_identity.example.id
  audience            = ["api://AzureADTokenExchange"]
  issuer              = azurerm_kubernetes_cluster.example.oidc_issuer_url
  subject             = "system:serviceaccount:default:azure-blob-proxy-demo"
}

## Assign role to workload identity
resource "azurerm_role_assignment" "example" {
  principal_id         = azurerm_user_assigned_identity.example.principal_id
  role_definition_name = "Storage Blob Data Reader"
  scope                = azurerm_storage_account.example.id
}
