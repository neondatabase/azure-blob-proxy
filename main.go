package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/azcore/log"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

var (
	listen        string = "127.0.0.1:8080"
	accountName   string
	containerName string

	// logger = slog.Default() // for default (unstructured) logs
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)

func main() {
	// parse comand line args and environment variables
	flag.StringVar(&listen, "http-listen-addr", LookupEnvOrString("HTTP_LISTEN_ADDR", listen), "http service listen address")
	flag.StringVar(&accountName, "account-name", LookupEnvOrString("AZURE_STORAGE_ACCOUNT", accountName), "Azure Storage account name")
	flag.StringVar(&containerName, "container-name", LookupEnvOrString("AZURE_STORAGE_CONTAINER", containerName), "Azure Storage blob container name")
	flag.Parse()

	// check mandatory parameters
	if len(accountName) == 0 {
		logger.Error("storage account name not specified")
		os.Exit(1)
	}
	if len(containerName) == 0 {
		logger.Error("storage container name not specified")
		os.Exit(1)
	}

	// print azure log output to stdout
	azlog.SetListener(func(event azlog.Event, s string) {
		logger.Info("azure log", "event", s)
	})
	// include only azidentity credential logs
	azlog.SetEvents(azidentity.EventAuthentication)

	// get azure credentials
	// https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/azidentity
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error("get credentials", "error", err)
		os.Exit(1)
	}

	// set azure blob storage client
	var client *azblob.Client
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	os.Setenv("AZURE_STORAGE_AUTH_MODE", "login")

	accountKey := LookupEnvOrString("AZURE_STORAGE_KEY", "")
	if len(accountKey) != 0 {
		// accountKey defined (running locally with azure-cli authentication)
		// use key to authorize
		key, keyerr := azblob.NewSharedKeyCredential(accountName, accountKey)
		if keyerr != nil {
			logger.Error("storage account key", "error", keyerr)
			os.Exit(1)
		}
		client, err = azblob.NewClientWithSharedKeyCredential(serviceURL, key, nil)
	} else {
		client, err = azblob.NewClient(serviceURL, cred, nil)
	}

	if err != nil {
		logger.Error("storage blob client", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// serve requests
	logger.Info("serving proxy requests", "addr", listen, "storage account", accountName, "container", containerName)
	http.Handle("/healthz", healthHandler(ctx))
	http.Handle("/", rootHandler(ctx, client, containerName))

	err = http.ListenAndServe(listen, nil)
	if err != nil {
		logger.Error("http listener", "error", err)
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
				logger.Error("download blob", "blob", blobFullName, "error", err)
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
