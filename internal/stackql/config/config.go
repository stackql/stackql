package config

import (
	"os"
	"path"
	"runtime"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/pkg/logging"
)

const defaultConfigCacheDir = ".stackql"

const defaultNixConfigCacheDirFileMode uint32 = 0755

const defaultWindowsConfigCacheDirFileMode uint32 = 0777

const defaultConfigFileName = ".stackqlrc"

const defaltLogLevel = "fatal"

const defaltErrorPresentation = "stderr"

const readlineDir = "readline"

const readlineTmpFile = "readline.tmp"

func GetDefaultLogLevelString() string {
	return defaltLogLevel
}

func GetDefaultErrorPresentationString() string {
	return defaltErrorPresentation
}

func GetWorkingDir() string {
	dir, err := os.Getwd()
	if err != nil {
		logging.GetLogger().Fatal(err)
	}
	return dir
}

func GetDefaultApplicationFilesRoot() string {
	return path.Join(GetWorkingDir(), defaultConfigCacheDir)
}

func GetDefaultConfigFilePath() string {
	return path.Join(GetWorkingDir(), defaultConfigFileName)
}

func GetReadlineDirPath(runtimeCtx dto.RuntimeCtx) string {
	return path.Join(runtimeCtx.ApplicationFilesRootPath, readlineDir)
}

func GetReadlineFilePath(runtimeCtx dto.RuntimeCtx) string {
	return path.Join(runtimeCtx.ApplicationFilesRootPath, readlineDir, readlineTmpFile)
}

func GetDefaultViperConfigFileName() string {
	return defaultConfigFileName
}

func GetDefaultKeyFilePath() string {
	return ""
}

func GetDefaultProviderCacheDirFileMode() uint32 {
	if runtime.GOOS == "windows" {
		return defaultWindowsConfigCacheDirFileMode
	}
	return defaultNixConfigCacheDirFileMode
}

func CreateDirIfNotExists(dirPath string, fileMode os.FileMode) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return os.Mkdir(dirPath, fileMode)
	}
	return nil
}
