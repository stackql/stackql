package config

import (
	"os"
	"path"
	"runtime"

	"github.com/stackql/stackql/internal/stackql/dto"

	log "github.com/sirupsen/logrus"
)

const defaultConfigCacheDir = ".stackql"

const defaultNixConfigCacheDirFileMode uint32 = 0755

const defaultWindowsConfigCacheDirFileMode uint32 = 0777

const defaultConfigFileName = ".iqlrc"

const defaultKeyFileName = "sa-key.json"

const defaltLogLevel = "warn"

const defaltErrorPresentation = "stderr"

const googleProvider = "google"

const oktaProvider = "okta"

const readlineDir = "readline"

const readlineTmpFile = "readline.tmp"

const defaultDbEngine = "sqlite3"

func GetGoogleProviderString() string {
	return googleProvider
}

func GetOktaProviderString() string {
	return oktaProvider
}

func GetDefaultLogLevelString() string {
	return defaltLogLevel
}

func GetDefaultErrorPresentationString() string {
	return defaltErrorPresentation
}

func GetWorkingDir() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func GetDefaultProviderCacheRoot() string {
	return path.Join(GetWorkingDir(), defaultConfigCacheDir)
}

func GetDefaultConfigFilePath() string {
	return path.Join(GetWorkingDir(), defaultConfigFileName)
}

func GetDefaultColorScheme() string {
	if runtime.GOOS == "windows" {
		return dto.DefaultWindowsColorScheme
	}
	return dto.DefaultColorScheme
}

func GetReadlineDirPath(runtimeCtx dto.RuntimeCtx) string {
	return path.Join(runtimeCtx.ProviderRootPath, readlineDir)
}

func GetReadlineFilePath(runtimeCtx dto.RuntimeCtx) string {
	return path.Join(runtimeCtx.ProviderRootPath, readlineDir, readlineTmpFile)
}

func GetDefaultViperConfigFileName() string {
	return defaultConfigFileName
}

func GetDefaultKeyFilePath() string {
	return ""
}

func GetDefaultDbEngine() string {
	return defaultDbEngine
}

func GetDefaultDbFilePath() string {
	return ""
}

func GetDefaultDbInitFilePath() string {
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
