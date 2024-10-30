package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

var (
	listen        string = "127.0.0.1:8080"
	accountName   string
	containerName string

	// Use logger = slog.Default() for default (unstructured) logs
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)

func main() {

	flag.StringVar(&listen, "http-listen-addr", LookupEnvOrString("HTTP_LISTEN_ADDR", listen), "http service listen address")
	flag.StringVar(&accountName, "account-name", LookupEnvOrString("AZURE_STORAGE_ACCOUNT_NAME", accountName), "Azure Storage account name")
	flag.StringVar(&containerName, "container-name", LookupEnvOrString("AZURE_STORAGE_CONTAINER_NAME", containerName), "Azure Storage blob container name")
	flag.Parse()

	if len(accountName) == 0 {
		logger.Error("Azure Storage account name not specified")
		os.Exit(1)
	}
	if len(containerName) == 0 {
		logger.Error("Azure Storage container name not specified")
		os.Exit(1)
	}

	// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error("can't obtain credentials", "error", err)
		os.Exit(1)
	}

	client, err := azblob.NewClient(fmt.Sprintf("https://%s.blob.core.windows.net/", accountName), cred, nil)
	if err != nil {
		logger.Error("can't create storage blob client", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// serve requests
	logger.Info("serving proxy requests", "addr", listen, "azure_storage_account", accountName, "container_name", containerName)
	http.Handle("/healthz", healthHandler(ctx))
	http.Handle("/", rootHandler(ctx, client, containerName))

	err = http.ListenAndServe(listen, nil)
	if err != nil {
		logger.Error("can't stat http listener", "error", err)
		os.Exit(1)
	}

} // main

func LookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func rootHandler(ctx context.Context, client *azblob.Client, containerName string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if key[0] == '/' {
			key = key[1:]
		}
		blobFullName := fmt.Sprintf("%s/%s", containerName, key)
		if r.Method == "GET" {
			streamResponse, err := client.DownloadStream(ctx, containerName, key, &azblob.DownloadStreamOptions{})
			if err != nil {
				logger.Error("blob not found", "blob", blobFullName)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			logger.Info("proxying", "blob", blobFullName)
			bufferedReader := bufio.NewReader(streamResponse.Body)
			_, err = bufferedReader.WriteTo(w)
			if err != nil {
				logger.Error("failed to proxy", "blob", blobFullName, "error", err)
			}
		} else {
			logger.Error("wrong method", "method", r.Method, "blob", blobFullName)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}
	return http.HandlerFunc(fn)
}

func healthHandler(ctx context.Context) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			return
		} else {
			logger.Error("wrong method", "method", r.Method, "path", r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}
	return http.HandlerFunc(fn)
}
