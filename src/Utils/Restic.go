package Utils

import "regexp"

var ResticOptionsKeepDaily = 90 // 90 days by default

var (
	ResticVersionReg     = regexp.MustCompile(`restic\s(\d+\.\d+\.\d+)\s.*`)
	ResticSnapshotReg    = regexp.MustCompile(`snapshot\s([0-9a-zA-Z]+)\ssaved`)
	ResticFileStatsReg   = regexp.MustCompile(`Files:\s*([0-9.]*) new,\s*([0-9.]*) changed,\s*([0-9.]*) unmodified`)
	ResticDirStatsReg    = regexp.MustCompile(`Dirs:\s*([0-9.]*) new,\s*([0-9.]*) changed,\s*([0-9.]*) unmodified`)
	ResticAddedBytesReg  = regexp.MustCompile(`Added to the repo: ([0-9.]+) (\w+)`)
	ResticProcessedReg   = regexp.MustCompile(`processed ([0-9.]*) files, ([0-9.]+) (\w+)`)
	ResticKeptSnapsReg   = regexp.MustCompile(`keep ([0-9.]*) snapshots:`)
	ResticRemoveSnapsReg = regexp.MustCompile(`remove ([0-9.]*) snapshots:`)
)

type ResticStats struct {
	FilesNew         int
	FilesChanged     int
	FilesUnmodified  int
	FilesProcessed   int
	DirsNew          int
	DirsChanged      int
	DirsUnmodified   int
	AddedToRepo      int
	BytesAdded       int
	BytesProcessed   int
	KeptSnapshots    int
	RemovedSnapshots int
}

func ConvertUnitRate(amount int, unit string) (int) {
        var result int64
	switch unit {
	case "TiB":
		result = int64(amount) * (1 << 40)
	case "GiB":
		result = int64(amount) * (1 << 30)
	case "MiB":
		result = int64(amount) * (1 << 20)
	case "KiB":
		result = int64(amount) * (1 << 10)
	}
	return int(result)
}
