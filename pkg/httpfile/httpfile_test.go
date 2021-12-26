package httpfile_test

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/xpy123993/toolbox/pkg/httpfile"
)

func createFileService(t *testing.T) (string, string) {
	tempDir := t.TempDir()
	server := httptest.NewServer(httpfile.CreateHTTPServiceMux(tempDir))
	t.Cleanup(server.Close)
	return server.URL, tempDir
}

func TestUpload(t *testing.T) {
	serverAddress, serveDir := createFileService(t)
	data := make([]byte, 64)
	if _, err := rand.Read(data); err != nil {
		t.Error(err)
	}

	anotherFolder := os.TempDir()
	if err := os.WriteFile(path.Join(anotherFolder, "test.bin"), data, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	if err := httpfile.UploadFile(fmt.Sprintf("%s/upload", serverAddress), path.Join(anotherFolder, "test.bin")); err != nil {
		t.Fatal(err)
	}
	if rdata, err := os.ReadFile(path.Join(serveDir, "test.bin")); err != nil {
		t.Fatal(err)
	} else if hex.EncodeToString(rdata) != hex.EncodeToString(data) {
		t.Errorf("data mismatched")
	}
}

func TestDownload(t *testing.T) {
	serverAddress, serveDir := createFileService(t)
	data := make([]byte, 64)
	if _, err := rand.Read(data); err != nil {
		t.Error(err)
	}

	if err := os.WriteFile(path.Join(serveDir, "test.bin"), data, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	anotherFolder := os.TempDir()
	if err := httpfile.DownloadFileFromURL(fmt.Sprintf("%s/download/test.bin", serverAddress), path.Join(anotherFolder, "test.bin")); err != nil {
		t.Fatal(err)
	}
	if rdata, err := os.ReadFile(path.Join(anotherFolder, "test.bin")); err != nil {
		t.Fatal(err)
	} else if hex.EncodeToString(rdata) != hex.EncodeToString(data) {
		t.Errorf("data mismatched: %s", rdata)
	}
}
