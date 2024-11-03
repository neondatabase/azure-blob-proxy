resource "azurerm_resource_group" "example" {
  name     = "azure-blob-proxy-demo"
  location = "eastus2"
}

resource "azurerm_storage_account" "example" {
  name                     = "azureblobproxydemo"
  resource_group_name      = azurerm_resource_group.example.name
  location                 = azurerm_resource_group.example.location
  account_tier             = "Standard"
  account_replication_type = "LRS"

  tags = {
    environment = "staging"
  }
}

resource "azurerm_storage_container" "example" {
  name                  = "azure-blob-proxy-demo"
  storage_account_name  = azurerm_storage_account.example.name
  container_access_type = "private"
}

resource "azurerm_storage_blob" "example" {
  name                   = "myfile.txt"
  storage_account_name   = azurerm_storage_account.example.name
  storage_container_name = azurerm_storage_container.example.name
  type                   = "Block"
  source_content         = "Hello, world"
}
