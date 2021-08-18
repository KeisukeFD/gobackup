package Utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type PrometheusLabels map[string]string

func CreatePrometheusMetric(name string, labels *[]PrometheusLabels, values []int) string {
	metricName := strings.ReplaceAll(name, "-", "_")
	metricName = strings.ReplaceAll(metricName, " ", "_")
	metricName = strings.ToLower(metricName)
	metricName = "backup_" + metricName

	helpText := fmt.Sprintf("# HELP %s Backup script collected metric", metricName)
	typeText := fmt.Sprintf("# TYPE %s gauge", metricName)

	var promLabels []string
	baseMetric := fmt.Sprintf("%s\n%s\n", helpText, typeText)
	var metric string
	for i, label := range *labels {
		promLabels = make([]string, 0)
		for k, v := range label {
			promLabels = append(promLabels, fmt.Sprintf("%s=\"%s\"", strings.ToLower(k), strings.ToLower(v)))
		}
		metric += fmt.Sprintf("%s{%s} %d\n",
			strings.ToLower(metricName),
			strings.Join(promLabels, ", "),
			values[i],
		)
	}
	return baseMetric + metric
}

func ExportMetricsToFile(filename string, metrics *[]string) error {
	f, err := ioutil.TempFile("/tmp/", "backup-metric-")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(f.Name())
		}
	}()

	bufw := bufio.NewWriter(f)

	for _, m := range *metrics {
		tmp := bytes.NewBufferString(m + "\n")
		bufw.Write(tmp.Bytes())
	}
	if err := bufw.Flush(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return moveFile(f.Name(), filename)
}

func moveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("Couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("Couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("Writing to output file failed: %s", err)
	}
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("Failed removing original file: %s", err)
	}
	return nil
}
