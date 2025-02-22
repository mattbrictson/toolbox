package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/semaphoreci/toolbox/cache-cli/pkg/archive"
	"github.com/semaphoreci/toolbox/cache-cli/pkg/files"
	"github.com/semaphoreci/toolbox/cache-cli/pkg/metrics"
	"github.com/semaphoreci/toolbox/cache-cli/pkg/storage"
	"github.com/semaphoreci/toolbox/cache-cli/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var storeCmd = &cobra.Command{
	Use:   "store [key path]",
	Short: "Store keys in the cache.",
	Long:  ``,
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		RunStore(cmd, args)
	},
}

func RunStore(cmd *cobra.Command, args []string) {
	if len(args) != 0 && len(args) != 2 {
		log.Error("Incorrect number of arguments!")
		_ = cmd.Help()
		return
	}

	storage, err := storage.InitStorage()
	utils.Check(err)

	metricsManager, err := metrics.InitMetricsManager(metrics.LocalBackend)
	utils.Check(err)

	archiver := archive.NewArchiver(metricsManager)

	if len(args) == 0 {
		lookupResults := files.Lookup(files.LookupOptions{
			GitBranch: FindGitBranch(),
			Restore:   false,
		})

		if len(lookupResults) == 0 {
			log.Info("Nothing to store in cache.")
			return
		}

		for _, lookupResult := range lookupResults {
			log.Infof("Detected %s.", lookupResult.DetectedFile)
			for _, entry := range lookupResult.Entries {
				log.Infof("Using default cache path '%s'.", entry.Path)
				key := entry.Keys[0]
				compressAndStore(storage, archiver, key, entry.Path)
			}
		}
	} else {
		path := filepath.FromSlash(args[1])
		compressAndStore(storage, archiver, args[0], path)
	}
}

func compressAndStore(storage storage.Storage, archiver archive.Archiver, rawKey, path string) {
	key := NormalizeKey(rawKey)
	if _, err := os.Stat(path); err == nil {
		if ok, _ := storage.HasKey(key); ok {
			log.Infof("Key '%s' already exists.", key)
			return
		}

		compressedFilePath, compressedFileSize, err := compress(archiver, key, path)
		if err != nil {
			log.Errorf("Error compressing %s: %v", path, err)
			return
		}

		maxSpace := storage.Config().MaxSpace
		if compressedFileSize > maxSpace {
			log.Errorf("Archive exceeds allocated %s for cache.", files.HumanReadableSize(maxSpace))
			return
		}

		uploadStart := time.Now()
		log.Infof("Uploading '%s' with cache key '%s'...", path, key)
		err = storage.Store(key, compressedFilePath)
		utils.Check(err)

		uploadDuration := time.Since(uploadStart)
		log.Infof("Upload complete. Duration: %v.", uploadDuration)

		err = os.Remove(compressedFilePath)
		if err != nil {
			log.Errorf("Error removing %s: %v", compressedFilePath, err)
		}
	} else {
		log.Infof("'%s' doesn't exist locally.", path)
	}
}

func compress(archiver archive.Archiver, key, path string) (string, int64, error) {
	compressingStart := time.Now()
	log.Infof("Compressing %s...", path)

	dst := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d", key, time.Now().Nanosecond()))
	err := archiver.Compress(dst, path)
	utils.Check(err)

	compressionDuration := time.Since(compressingStart)
	info, err := os.Stat(dst)
	if err != nil {
		_ = os.Remove(dst)
		return "", -1, err
	}

	log.Infof("Compression complete. Duration: %v. Size: %v bytes.", compressionDuration.String(), files.HumanReadableSize(info.Size()))
	return dst, info.Size(), nil
}

func NormalizeKey(key string) string {
	normalizedKey := strings.ReplaceAll(key, "/", "-")
	if normalizedKey != key {
		log.Infof("Key '%s' is normalized to '%s'.", key, normalizedKey)
	}

	return normalizedKey
}

func FindGitBranch() string {
	gitPrBranch := os.Getenv("SEMAPHORE_GIT_PR_BRANCH")
	if gitPrBranch != "" {
		return gitPrBranch
	}

	return os.Getenv("SEMAPHORE_GIT_BRANCH")
}

func init() {
	RootCmd.AddCommand(storeCmd)
}
