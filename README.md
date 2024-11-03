[![build](https://github.com/neondatabase/azure-blob-proxy/actions/workflows/build.yml/badge.svg)](https://github.com/neondatabase/azure-blob-proxy/actions/workflows/build.yml)

# Azure blob proxy

Simple read-only access to Azure Blob storage containers

# Local run

> [!NOTE]
> Learn how to create, load, and list blobs using the Azure CLI in the [official docs](https://learn.microsoft.com/en-us/azure/storage/blobs/storage-quickstart-blobs-cli)

## Prepare

### authenticate to Azure

```console
az login
```

### set environment variables

```console
export AZURE_LOCATION=eastus2
export AZURE_RESOURCE_GROUP=azure-blob-proxy-demo
export AZURE_STORAGE_ACCOUNT=azureblobproxydemo
export AZURE_STORAGE_CONTAINER=azure-blob-proxy-demo
```

### create Azure resource group

```console
az group create \
    --name $AZURE_RESOURCE_GROUP \
    --location $AZURE_LOCATION
```

### create Azure stroage account

```console
az storage account create \
    --name $AZURE_STORAGE_ACCOUNT \
    --resource-group $AZURE_RESOURCE_GROUP \
    --location $AZURE_LOCATION \
    --sku Standard_ZRS
```

### export storage account access key

```console
export AZURE_STORAGE_KEY=$(az storage account keys list --account-name $AZURE_STORAGE_ACCOUNT | jq -r '.[0].value')
```

### create a container in a storage account

```console
az storage container create \
    --name $AZURE_STORAGE_CONTAINER \
    --account-name $AZURE_STORAGE_ACCOUNT
```

### create demo file

```console
echo "Hello, world" > myfile.txt
```

### upload file to container

```console
az storage blob upload \
    --account-name $AZURE_STORAGE_ACCOUNT \
    --container-name $AZURE_STORAGE_CONTAINER \
    --name myfile.txt \
    --file myfile.txt
```

### check (list) that file present as blob in the storage

```console
az storage blob list \
    --account-name $AZURE_STORAGE_ACCOUNT \
    --container-name $AZURE_STORAGE_CONTAINER \
    --output table
```

## Run

### start azure-blob-proxy

```console
go run main.go
```

### in another terminal check how the proxy works

```console
for i in 1 2 3; do curl 127.0.0.1:8080/myfile.txt; done
```
you should see file content

```console
Hello, world
Hello, world
Hello, world
```

### inspect azure-blob-proxy logs

blob (myfile.txt) successfully proxied

```josn
{"time":"2024-11-03T07:30:40.548932+08:00","level":"INFO","msg":"serving proxy requests","addr":"127.0.0.1:8080","azure_storage_account":"azureblobproxydemo","container_name":"azure-blob-proxy-demo"}
{"time":"2024-11-03T07:31:11.772515+08:00","level":"INFO","msg":"proxying","blob":"azure-blob-proxy-demo/myfile.txt"}
{"time":"2024-11-03T07:31:12.025935+08:00","level":"INFO","msg":"proxying","blob":"azure-blob-proxy-demo/myfile.txt"}
{"time":"2024-11-03T07:31:12.263774+08:00","level":"INFO","msg":"proxying","blob":"azure-blob-proxy-demo/myfile.txt"}
```

## Tear down

```console
rm -f myfile.txt

az group delete --name $AZURE_RESOURCE_GROUP

az logout
```

```console
unset AZURE_LOCATION
unset AZURE_RESOURCE_GROUP
unset AZURE_STORAGE_ACCOUNT
unset AZURE_STORAGE_CONTAINER
unset AZURE_STORAGE_KEY
```
