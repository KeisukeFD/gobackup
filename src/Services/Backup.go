package Services

import (
	"fmt"
	"gobackup/src/Model"
	"gobackup/src/Utils"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BackupManager struct {
	Config      *Model.Config
	StepResults []BackupStepResult
	LastResult  *BackupStepResult
}

type BackupStatus int

const (
	Success BackupStatus = 0
	Failed  BackupStatus = 1
)

func (s BackupStatus) String() string {
	switch s {
	case Success:
		return "success"
	case Failed:
		return "failed"
	}
	return "unknown"
}

type BackupStepResult struct {
	Name      string
	ShortName string
	Status    BackupStatus
	Output    string
	Duration  time.Duration
}

var backup *BackupManager

func InitBackupManager(config *Model.Config, repository string, folders []string) *BackupManager {
	var once sync.Once
	once.Do(func() {
		backup = &BackupManager{}
		backup.Config = config
		backup.Config.Repository = repository
		backup.Config.FoldersToBackup = folders
	})
	return backup
}

func GetBackupManager() *BackupManager {
	return backup
}

func (bm *BackupManager) ExecutePreCommand() (string, error) {
	cmd := bm.Config.BackupConfig.Backup.PreExecution
	if cmd != "" {
		res, err := Utils.ExecuteCommand(cmd)
		return res.Output, err
	}
	return "", nil
}

func (bm *BackupManager) ExecutePostCommand() (string, error) {
	cmd := bm.Config.BackupConfig.Backup.PostExecution
	if cmd != "" {
		res, err := Utils.ExecuteCommand(cmd)
		return res.Output, err
	}
	return "", nil
}

func (bm *BackupManager) InitRepo() {
	result := BackupStepResult{
		Name:      "Initialize Repository",
		ShortName: "InitRepo",
	}
	Utils.GetLogger().Info(result.Name)
	startTime := time.Now()
	res, _ := bm.ExecuteRestic("init")
	result.Output = res.Output
	result.Status = Success

	if res.ExitCode == 0 || (res.ExitCode == 1 && strings.Contains(res.Output, "already exists")) {
		result.Status = Success
	} else {
		result.Status = Failed
	}
	endTime := time.Now()
	result.Duration = endTime.Sub(startTime)
	bm.StepResults = append(bm.StepResults, result)
	bm.LastResult = &result
}

func (bm *BackupManager) StartBackup() {
	result := BackupStepResult{
		Name:      "Backing up",
		ShortName: "StartBackup",
	}
	Utils.GetLogger().Info(result.Name)
	if !isLastResultSuccess(bm.LastResult) {
		return
	}
	startTime := time.Now()
	var options string
	if bm.Config.BackupConfig.Information.ExclusionFile != "" {
		options += "--exclude-file=" + bm.Config.BackupConfig.Information.ExclusionFile
	}
	options += fmt.Sprintf(" --tag=%s", bm.Config.BackupConfig.Information.ServerName)

	var foldersToBackup []string
	for _, f := range bm.Config.FoldersToBackup {
		if strings.Contains(f, " ") {
			foldersToBackup = append(foldersToBackup, fmt.Sprintf(`'%s'`, f))
		} else {
			foldersToBackup = append(foldersToBackup, f)
		}
	}

	cmd := createBashCommand(
		"backup",
		options,
		strings.Join(foldersToBackup, " "),
	)
	res, _ := bm.ExecuteRestic(cmd)
	result.Output = res.Output
	result.Status = Success

	if snapshotId := Utils.ResticSnapshotReg.FindStringSubmatch(res.Output); len(snapshotId) == 0 {
		result.Status = Failed
	}
	if res.ExitCode != 0 {
		result.Status = Failed
	}
	endTime := time.Now()
	result.Duration = endTime.Sub(startTime)
	bm.StepResults = append(bm.StepResults, result)
	bm.LastResult = &result
}

func (bm *BackupManager) Cleanup() {
	result := BackupStepResult{
		Name:      "Cleanup Repository",
		ShortName: "Cleanup",
	}
	Utils.GetLogger().Info(result.Name)
	if !isLastResultSuccess(bm.LastResult) {
		return
	}
	startTime := time.Now()

	var options string
	if len(bm.Config.BackupConfig.ResticOptions) > 0 {
		options += strings.Join(bm.Config.BackupConfig.ResticOptions, " ")
	} else {
		options += strings.Join(Utils.DefaultResticOptions, " ")
	}
	options += fmt.Sprintf(" --tag=%s", bm.Config.BackupConfig.Information.ServerName)

	cmd := createBashCommand(
		"forget",
		options,
		"--prune",
		"-c",
	)

	res, _ := bm.ExecuteRestic(cmd)
	result.Status = Success
	result.Output = res.Output

	if res.ExitCode != 0 {
		result.Status = Failed
	}
	endTime := time.Now()
	result.Duration = endTime.Sub(startTime)
	bm.StepResults = append(bm.StepResults, result)
	bm.LastResult = &result
}

func (bm *BackupManager) CheckRepoIntegrity() {
	result := BackupStepResult{
		Name:      "Check Repository Integrity",
		ShortName: "CheckRepoIntegrity",
	}
	Utils.GetLogger().Info(result.Name)
	if !isLastResultSuccess(bm.LastResult) {
		return
	}
	startTime := time.Now()
	res, _ := bm.ExecuteRestic("check")
	result.Output = res.Output
	result.Status = Success

	if res.ExitCode != 0 {
		result.Status = Failed
	}
	if strings.Contains(res.Output, "no errors were found") {
		result.Status = Success
	}

	endTime := time.Now()
	result.Duration = endTime.Sub(startTime)
	bm.StepResults = append(bm.StepResults, result)
	bm.LastResult = &result
}

func (bm *BackupManager) GetResults() (BackupStatus, *[]BackupStepResult) {
	finalStatus := getFinalStatus(bm.StepResults)
	if finalStatus == Success {
		Utils.GetLogger().Info("Backup finished successfully !")
	} else {
		Utils.GetLogger().Warning("Backup failed !")
	}
	return finalStatus, nil
}

func (bm *BackupManager) GetMetrics() *[]string {
	resticStats := &Utils.ResticStats{}
	defaultLabels := &Utils.PrometheusLabels{
		"repository": bm.Config.Repository,
		"client":     bm.Config.BackupConfig.Information.ClientName,
		"name":       bm.Config.BackupConfig.Information.ServerName,
	}
	for _, res := range bm.StepResults {
		if strings.ToLower(res.ShortName) == "startbackup" {
			if filesStats := Utils.ResticFileStatsReg.FindStringSubmatch(res.Output); len(filesStats) == 4 {
				if tmp, err := strconv.Atoi(filesStats[1]); err == nil {
					resticStats.FilesNew = tmp
				}
				if tmp, err := strconv.Atoi(filesStats[2]); err == nil {
					resticStats.FilesChanged = tmp
				}
				if tmp, err := strconv.Atoi(filesStats[3]); err == nil {
					resticStats.FilesUnmodified = tmp
				}
			}
			if dirStats := Utils.ResticDirStatsReg.FindStringSubmatch(res.Output); len(dirStats) == 4 {
				if tmp, err := strconv.Atoi(dirStats[1]); err == nil {
					resticStats.DirsNew = tmp
				}
				if tmp, err := strconv.Atoi(dirStats[2]); err == nil {
					resticStats.DirsChanged = tmp
				}
				if tmp, err := strconv.Atoi(dirStats[3]); err == nil {
					resticStats.DirsUnmodified = tmp
				}
			}
			if addedStats := Utils.ResticAddedBytesReg.FindStringSubmatch(res.Output); len(addedStats) == 3 {
				if tmp, err := strconv.ParseFloat(addedStats[1], 64); err == nil {
					tmp *= 1000
					resticStats.BytesProcessed = Utils.ConvertUnitRate(int(tmp), addedStats[2])
				}
			}

			if processedStats := Utils.ResticProcessedReg.FindStringSubmatch(res.Output); len(processedStats) == 4 {
				if tmp, err := strconv.Atoi(processedStats[1]); err == nil {
					resticStats.FilesProcessed = tmp
				}
				if tmp, err := strconv.ParseFloat(processedStats[2], 64); err == nil {
					tmp *= 1000
					resticStats.BytesProcessed = Utils.ConvertUnitRate(int(tmp), processedStats[3])
				}
			}
		} else if strings.ToLower(res.ShortName) == "cleanup" {
			if keepSnapshots := Utils.ResticKeptSnapsReg.FindStringSubmatch(res.Output); len(keepSnapshots) == 2 {
				if tmp, err := strconv.Atoi(keepSnapshots[1]); err == nil {
					resticStats.KeptSnapshots = tmp
				}
			}
			if removeSnapshots := Utils.ResticRemoveSnapsReg.FindStringSubmatch(res.Output); len(removeSnapshots) == 2 {
				if tmp, err := strconv.Atoi(removeSnapshots[1]); err == nil {
					resticStats.RemovedSnapshots = tmp
				}
			}
		}

	}
	metrics := make([]string, 0)
	metrics = append(metrics,
		Utils.CreatePrometheusMetric("files_stats", &[]Utils.PrometheusLabels{
			Utils.MergeMap(Utils.PrometheusLabels{
				"type":   "files",
				"action": "new",
			}, *defaultLabels),
			Utils.MergeMap(Utils.PrometheusLabels{
				"type":   "files",
				"action": "changed",
			}, *defaultLabels),
			Utils.MergeMap(Utils.PrometheusLabels{
				"type":   "files",
				"action": "unmodified",
			}, *defaultLabels),
			Utils.MergeMap(Utils.PrometheusLabels{
				"type":   "directories",
				"action": "new",
			}, *defaultLabels),
			Utils.MergeMap(Utils.PrometheusLabels{
				"type":   "directories",
				"action": "changed",
			}, *defaultLabels),
			Utils.MergeMap(Utils.PrometheusLabels{
				"type":   "directories",
				"action": "unmodified",
			}, *defaultLabels),
		}, []int{
			resticStats.FilesNew,
			resticStats.FilesChanged,
			resticStats.FilesUnmodified,
			resticStats.DirsNew,
			resticStats.DirsChanged,
			resticStats.DirsUnmodified,
		}),
	)
	metrics = append(metrics,
		Utils.CreatePrometheusMetric("bytes_processed",
			&[]Utils.PrometheusLabels{
				Utils.MergeMap(nil, *defaultLabels),
			},
			[]int{
				resticStats.BytesProcessed,
			}),
	)
	metrics = append(metrics,
		Utils.CreatePrometheusMetric("bytes_added",
			&[]Utils.PrometheusLabels{
				Utils.MergeMap(nil, *defaultLabels),
			},
			[]int{
				resticStats.BytesAdded,
			}),
	)
	metrics = append(metrics,
		Utils.CreatePrometheusMetric("snapshots",
			&[]Utils.PrometheusLabels{
				Utils.MergeMap(Utils.PrometheusLabels{
					"action": "keep",
				}, *defaultLabels),
				Utils.MergeMap(Utils.PrometheusLabels{
					"action": "removed",
				}, *defaultLabels),
			},
			[]int{
				resticStats.KeptSnapshots,
				resticStats.RemovedSnapshots,
			}),
	)
	finalStatus := getFinalStatus(bm.StepResults)
	metricStatus := 0
	if finalStatus == Success {
		metricStatus = 1
	}
	metrics = append(metrics,
		Utils.CreatePrometheusMetric("status",
			&[]Utils.PrometheusLabels{
				Utils.MergeMap(Utils.PrometheusLabels{
					"status": finalStatus.String(),
				}, *defaultLabels),
			},
			[]int{
				metricStatus,
			}),
	)
	return &metrics
}

func (bm *BackupManager) MakeEmailBody() string {
	var totalDuration time.Duration
	startTime := time.Now().Format("2006-06-02 15:04:05")

	backupName := createPathName(
		bm.Config.BackupConfig.Information.ClientName,
		bm.Config.BackupConfig.Information.ServerName,
		bm.Config.Repository,
	)

	body := fmt.Sprintf("Subject: [%s] Backup '%s' - %s\n\n",
		strings.Title(getFinalStatus(bm.StepResults).String()),
		backupName[:len(backupName)-1],
		startTime,
	)
	for _, res := range bm.StepResults {
		titleSize := len(res.Name) + 6
		body += strings.Repeat("#", titleSize) + "\n"
		body += "# " + res.Name + "\n"
		body += strings.Repeat("#", titleSize) + "\n"
		body += res.Output
		body += "Duration: "
		body += Utils.HumanDuration(res.Duration.Seconds()) + "\n\n"
		totalDuration += res.Duration
	}
	body += "Total Duration: "
	body += Utils.HumanDuration(totalDuration.Seconds()) + "\n"

	finalStatus := getFinalStatus(bm.StepResults)
	if finalStatus == Success {
		body += "Backup finished successfully !"
	} else {
		body += "Backup failed !"
	}
	return body
}

func (bm *BackupManager) ExecuteRestic(command string) (Utils.CommandResult, error) {
	pathName := createPathName(
		bm.Config.BackupConfig.Information.ClientName,
		bm.Config.BackupConfig.Information.ServerName,
		bm.Config.Repository,
	)
	repository := "rclone:" + bm.Config.BackupConfig.Information.RCloneConnectionName + ":" + createPathName(
		bm.Config.BackupConfig.Information.BucketName,
		pathName,
	)

	cmd := createBashCommand(
		bm.Config.BackupConfig.Binaries.Restic,
		"-r "+repository,
		command,
	)
	Utils.GetLogger().Debug(cmd)
	envs := make(map[string]string)
	envs["RESTIC_PASSWORD"] = bm.Config.ResticPassword
	result, err := Utils.ExecuteCommandWithEnv(cmd, envs)
	if result.Output != "" {
		Utils.GetLogger().Debug(result.Output)
	}
	return result, err
}

/**** Private ****/
/*****************/
func createBashCommand(options ...string) string {
	return strings.Join(options, " ")
}

func createPathName(paths ...string) string {
	var path string
	for _, p := range paths {
		if p != "" {
			path += p
		}
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
	}
	return path
}

func isLastResultSuccess(result *BackupStepResult) bool {
	if result != nil && result.Status == Failed {
		Utils.GetLogger().Warning("Error in the step: " + result.Name + ", bypassing current step.")
		return false
	}
	return true
}

func getFinalStatus(results []BackupStepResult) BackupStatus {
	var finalStatus BackupStatus
	for _, result := range results {
		finalStatus = result.Status
		if result.Status == Failed {
			break
		}
	}
	return finalStatus
}
