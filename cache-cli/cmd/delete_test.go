package cmd

import (
	"io/ioutil"
	"testing"

	"github.com/semaphoreci/toolbox/cache-cli/pkg/storage"
	"github.com/semaphoreci/toolbox/cache-cli/pkg/utils"
	assert "github.com/stretchr/testify/assert"
)

func Test__Delete(t *testing.T) {
	storage, err := storage.InitStorage()
	assert.Nil(t, err)

	t.Run("key is missing", func(*testing.T) {
		capturer := utils.CreateOutputCapturer()
		RunDelete(hasKeyCmd, []string{"this-key-does-not-exist"})
		output := capturer.Done()

		assert.Contains(t, output, "The key 'this-key-does-not-exist' doesn't exist in the cache store.")
	})

	t.Run("key is present", func(*testing.T) {
		storage.Clear()
		tempFile, _ := ioutil.TempFile("/tmp", "*")
		storage.Store("abc001", tempFile.Name())

		capturer := utils.CreateOutputCapturer()
		RunDelete(hasKeyCmd, []string{"abc001"})
		output := capturer.Done()

		assert.Contains(t, output, "Key 'abc001' is deleted.")
	})
}