package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/semaphoreci/toolbox/cache-cli/pkg/storage"
	"github.com/semaphoreci/toolbox/cache-cli/pkg/utils"
	assert "github.com/stretchr/testify/assert"
)

func Test__Delete(t *testing.T) {
	runTestForAllBackends(t, func(backend string, storage storage.Storage) {
		t.Run(fmt.Sprintf("%s key is missing", backend), func(*testing.T) {
			capturer := utils.CreateOutputCapturer()
			RunDelete(deleteCmd, []string{"this-key-does-not-exist"})
			output := capturer.Done()

			assert.Contains(t, output, "Key 'this-key-does-not-exist' doesn't exist in the cache store.")
		})

		t.Run(fmt.Sprintf("%s key is present", backend), func(*testing.T) {
			storage.Clear()
			tempFile, _ := ioutil.TempFile(os.TempDir(), "*")
			storage.Store("abc001", tempFile.Name())

			capturer := utils.CreateOutputCapturer()
			RunDelete(deleteCmd, []string{"abc001"})
			output := capturer.Done()

			assert.Contains(t, output, "Key 'abc001' is deleted.")
		})

		t.Run(fmt.Sprintf("%s normalizes key", backend), func(*testing.T) {
			storage.Clear()
			tempFile, _ := ioutil.TempFile(os.TempDir(), "*")
			RunStore(NewStoreCommand(), []string{"abc/00/33", tempFile.Name()})

			capturer := utils.CreateOutputCapturer()
			RunDelete(deleteCmd, []string{"abc/00/33"})
			output := capturer.Done()

			assert.Contains(t, output, "Key 'abc/00/33' is normalized to 'abc-00-33'")
			assert.Contains(t, output, "Key 'abc-00-33' is deleted.")
		})
	})
}
