package store

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/semaphoreci/toolbox/sem-context/pkg/utils"
)

const keysInfoDirName = ".workflow-context/"

func Put(key, value string) error {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return &utils.Error{ErrorMessage: "Cant create temp file to store contents from artifacts", ExitCode: 2}
	}
	defer os.Remove(file.Name())
	file.Write([]byte(value))

	// TODO this logic should not be in here, but rather in command
	// existing_value, _ := Get(key)
	// if existing_value != "" && !flags.Force {
	// 	utils.CheckError(fmt.Errorf("Key with same name already exists. Delete existing key or use --force flag"), 1)
	// }
	contextId := utils.GetPipelineContextHierarchy()[0]
	artifact_output, err := execArtifactCommand(Push, file.Name(), keysInfoDirName+contextId+"/"+key)
	if err != nil {
		log.New(os.Stderr, "", 0).Println(artifact_output)
		return &utils.Error{ErrorMessage: "Cant execute artifacts push command to store key-value pair", ExitCode: 2}
	}

	//Since the key is stored, delete it from '.deleted' dir, in case it was marked as deleted before
	execArtifactCommand(Yank, keysInfoDirName+contextId+"/.deleted/"+key, "")
	fmt.Fprintf(os.Stdout, "Key-value pair successfully stored")
	return nil
}

func Get(key string) (string, error) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return "", &utils.Error{ErrorMessage: "Cant create temp file to store contents from artifacts", ExitCode: 2}
	}
	defer os.Remove(file.Name())

	contextHierarchy := utils.GetPipelineContextHierarchy()
	found_the_key := false
	for _, contextID := range contextHierarchy {
		_, err = execArtifactCommand(Pull, keysInfoDirName+contextID+"/"+key, file.Name())
		if err == nil {
			found_the_key = true
			break
		}

		//If key is deleted, we dont need to go looking for it in parent contexts
		key_deleted, err := checkIfKeyDeleted(contextID, key)
		if err != nil {
			return "", err
		}
		if key_deleted {
			break
		}
	}

	if !found_the_key {
		return "", &utils.Error{ErrorMessage: fmt.Sprintf("Cant find the key '%s'", key), ExitCode: 1}
	}

	byte_key, _ := os.ReadFile(file.Name())
	return string(byte_key), nil
}

func Delete(key string) error {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return &utils.Error{
			ErrorMessage: "Cant create temp file needed to perform delete operation when using artifacts as store",
			ExitCode:     2,
		}
	}
	defer os.Remove(file.Name())

	contextId := utils.GetPipelineContextHierarchy()[0]
	execArtifactCommand(Yank, keysInfoDirName+contextId+"/"+key, "")
	// The key might be present in some of the parent pipline's context as well, but we cant delete them there, as they might be used by some other pipeline.
	// We will just mark those keys as deleted inside this pipeline's context.
	artifact_output, err := execArtifactCommand(Push, file.Name(), keysInfoDirName+contextId+"/.deleted/"+key)
	if err != nil {
		// Since 'artifact' CLI always returns 1, this is the only way to check if
		// communication with artifact server is the problem, of key just does not exist
		if !strings.Contains(artifact_output, "Artifact not found") {
			log.New(os.Stderr, "", 0).Panicln(artifact_output)
			return &utils.Error{ErrorMessage: "Error with establishing connection with artifact server", ExitCode: 2}
		}
	}
	return nil
}

type ArtifactCommand string

const (
	Push ArtifactCommand = "push"
	Pull                 = "pull"
	Yank                 = "yank"
)

func checkIfKeyDeleted(contextID, key string) (bool, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return false, &utils.Error{
			ErrorMessage: "Cant create temp file needed which is needed when using artifacts as store",
			ExitCode:     2,
		}
	}
	defer os.RemoveAll(dir)

	//TODO check what this function returns
	execArtifactCommand(Pull, keysInfoDirName+contextID+"/.deleted/", dir)

	all_deleted_key_files, _ := ioutil.ReadDir(dir)
	for _, deleted_key_file := range all_deleted_key_files {
		if key == deleted_key_file.Name() {
			return true, nil
		}
	}
	return false, nil
}

func execArtifactCommand(command ArtifactCommand, source, dest string) (string, error) {
	var cmd *exec.Cmd
	if command == Push || command == Pull {
		cmd = exec.Command("artifact", fmt.Sprintf("%v", command), "workflow", source, "-d", dest, "--force")
	} else {
		cmd = exec.Command("artifact", fmt.Sprintf("%v", command), "workflow", source)
	}
	artifact_output, err := cmd.CombinedOutput()
	return string(artifact_output), err
}