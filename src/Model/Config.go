package Model

import (
	"fmt"
	"gobackup/src/Utils"
	"golang.org/x/term"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"sync"
	"syscall"
)

type Config struct {
	Environment     string
	LoggerLevel     string
	GinMode         string
	BackupConfig    *BackupConfig
	ResticPassword  string
	FoldersToBackup []string
	Repository      string
}

type BackupConfig struct {
	Information struct {
		ClientName           string `yaml:"client_name"`
		ServerName           string `yaml:"server_name" required:"true"`
		RCloneConnectionName string `yaml:"rclone_connection_name" required:"true"`
		BucketName           string `yaml:"bucket_name" required:"true"`
		ExclusionFile        string `yaml:"exclusion_file"`
		KeepDaily            int    `yaml:"keep_daily"`
	} `yaml:"information"`
	Binaries struct {
		Restic string `yaml:"restic"  required:"true"`
	} `yaml:"binaries"`
	Email struct {
		Enabled  bool   `yaml:"enabled"`
		Sender   string `yaml:"sender"`
		Password string `yaml:"password"`
		To       string `yaml:"to"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		MaxTry   int    `yaml:"max_try"`
	} `yaml:"email"`
	Backup struct {
		PreExecution  string `yaml:"pre_exec"`
		PostExecution string `yaml:"post_exec"`
	} `yaml:"backup"`
}

var instance *Config

func GetConfig() *Config {
	if instance != nil {
		return instance
	}
	var once sync.Once
	once.Do(func() {
		cfg, err := getConfigEnv()
		if err != nil {
			fmt.Println("Error loading configuration: ", err)
			os.Exit(1)
		}
		instance = cfg
	})
	return instance
}

func getConfigEnv() (*Config, error) {
	cfg := &Config{}

	if _, ok := os.LookupEnv("LOG_LEVEL"); !ok {
		cfg.LoggerLevel = "INFO"
	} else {
		cfg.LoggerLevel = os.Getenv("LOG_LEVEL")
	}

	if _, ok := os.LookupEnv("ENV"); !ok {
		cfg.Environment = "PROD"
	} else {
		cfg.Environment = os.Getenv("ENV")
	}

	switch strings.ToLower(cfg.Environment) {
	case "dev":
		cfg.LoggerLevel = "debug"
		cfg.GinMode = "debug"
	default:
		cfg.GinMode = "release"
	}

	return cfg, nil
}

func (c *Config) InitBackupConfig(filename string) {
	Utils.GetLogger().Debug("Loading configuration '", filename, "'")

	backupConfig := &c.BackupConfig
	yamlFile, err := ioutil.ReadFile(filename)
	Utils.HaltOnError(Utils.GetLogger(), err, "Impossible to open file '"+filename+"'")
	err = yaml.UnmarshalStrict(yamlFile, backupConfig)
	Utils.HaltOnError(Utils.GetLogger(), err, "Error parsing yaml")

	c.validateBackupConfiguration()
}

func (c *Config) GetResticPassword() {
	if passwd, ok := os.LookupEnv("RESTIC_PASSWORD"); !ok || passwd == "" {
		fmt.Println("Restic password: ")
		password, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			fmt.Println("Error: Impossible to get Restic Password")
			os.Exit(1)
		}
		c.ResticPassword = string(password)
	} else {
		c.ResticPassword = os.Getenv("RESTIC_PASSWORD")
	}
}

func (c *Config) validateBackupConfiguration() {
	Utils.GetLogger().Debug("Checking configuration")
	requiredFields := make(map[string]string)
	configIsValid := _checkRequiredFields(c.BackupConfig, func(field string, yamlField string) {
		requiredFields[field] = yamlField
	})

	if !configIsValid {
		Utils.GetLogger().Error("Some fields are required in the configuration !")
		for k := range requiredFields {
			Utils.GetLogger().Warning("- ", k, " (", requiredFields[k], ")")
		}
		os.Exit(1)
	}

	_, err := os.Stat(c.BackupConfig.Binaries.Restic)
	Utils.HaltOnError(Utils.GetLogger(), err, "")

	result, err := Utils.ExecuteCommand(c.BackupConfig.Binaries.Restic + " version")
	Utils.GetLogger().Debug("Version: ", result.Output)
	Utils.HaltOnError(Utils.GetLogger(), err, "")

	if resticVersions := Utils.ResticVersionReg.FindStringSubmatch(result.Output); len(resticVersions) == 0 {
		Utils.HaltOnError(Utils.GetLogger(), nil, "Can't find restic")
	} else {
		Utils.GetLogger().Info("Restic version " + resticVersions[1] + " found !")
	}

	if c.BackupConfig.Information.ExclusionFile != "" {
		_, err := os.Stat(c.BackupConfig.Information.ExclusionFile)
		Utils.HaltOnError(Utils.GetLogger(), err, "Does the exclusion file exists ?")
	}
}

func _checkRequiredFields(backupConfig *BackupConfig, callback func(string, string)) bool {
	isValid := true
	bckCfgType := reflect.ValueOf(*backupConfig)

	for i := 0; i < bckCfgType.NumField(); i++ {
		if bckCfgType.Type().Field(i).Type.Kind() == reflect.Struct {
			tmp := reflect.ValueOf(bckCfgType.Field(i).Interface())
			for j := 0; j < tmp.NumField(); j++ {
				if tmp.Type().Field(j).Tag.Get("required") == "true" && tmp.Field(j).String() == "" {
					callback(
						bckCfgType.Type().Field(i).Name+"->"+tmp.Type().Field(j).Name,
						bckCfgType.Type().Field(i).Tag.Get("yaml")+"."+tmp.Type().Field(j).Tag.Get("yaml"),
					)
					isValid = false
				}
			}
		}
	}
	return isValid
}
